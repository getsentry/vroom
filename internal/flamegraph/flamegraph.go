package flamegraph

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/storageutil"
)

type Flamegraph struct {
	stacks   [][]frame.Frame
	counters map[uint64]int
}

func (f *Flamegraph) AddStack(stack []frame.Frame, count int) {
	fingerprint := stack[len(stack)-1].Fingerprint
	if _, exists := (*f).counters[fingerprint]; exists {
		f.counters[fingerprint] += count
	} else {
		f.counters[fingerprint] = count
		cp := make([]frame.Frame, len(stack))
		copy(cp, stack)
		f.stacks = append(f.stacks, cp)
	}
}

func ConvertStackTracesToFlamegraph(
	flamegraph *Flamegraph,
	minFreq int) speedscope.Output {

	// filter out stack traces with a frequency less
	// than minFreq
	n := 0
	for _, stack := range (*flamegraph).stacks {
		if (*flamegraph).counters[stack[len(stack)-1].Fingerprint] >= minFreq {
			(*flamegraph).stacks[n] = stack
			n++
		}
	}
	(*flamegraph).stacks = (*flamegraph).stacks[:n]

	var frames []speedscope.Frame
	samples := make([][]int, 0, len((*flamegraph).stacks))
	addressToIndex := make(map[string]int)
	weights := make([]uint64, 0, len((*flamegraph).stacks))
	var endValue uint64 = 0

	for _, stack := range (*flamegraph).stacks {
		weight := (*flamegraph).counters[stack[len(stack)-1].Fingerprint]
		endValue += uint64(weight)
		sample := make([]int, 0, len(stack))

		for _, frame := range stack {
			frameAddress := getFrameID(frame)
			if index, exist := addressToIndex[frameAddress]; exist {
				sample = append(sample, index)
			} else {
				addressToIndex[frameAddress] = len(frames)
				sample = append(sample, len(frames))
				frames = append(frames, speedscope.Frame{
					Name:  frame.Function,
					Image: frame.PackageBaseName(),
					Path:  frame.Path,
				})
			}
		}
		samples = append(samples, sample)
		weights = append(weights, uint64(weight))
	}

	aggProfiles := make([]interface{}, 1)
	aggProfiles[0] = speedscope.SampledProfile{
		Samples:      samples,
		Weights:      weights,
		IsMainThread: true,
		Type:         speedscope.ProfileTypeSampled,
		Unit:         speedscope.ValueUnitCount,
		EndValue:     endValue,
	}

	return speedscope.Output{
		Shared: speedscope.SharedData{
			Frames: frames,
		},
		Profiles: aggProfiles,
	}
}

func ProcessStacksFromCallTrees(callTrees map[uint64][]*nodetree.Node, f *Flamegraph) {

	for _, threadTrees := range callTrees {
		for _, tree := range threadTrees {
			// 128 is the max stack size
			currentStack := make([]frame.Frame, 0, 128)
			visitTree(f, tree, &currentStack)
		}
	}
}

func visitTree(f *Flamegraph, node *nodetree.Node, currentStack *[]frame.Frame) {
	currentFrame := node.Frame()
	*currentStack = append(*currentStack, currentFrame)

	// base case (when we reach leaf frames)
	if node.Children == nil {
		f.AddStack(*currentStack, node.SampleCount)
	} else {
		totChildrenSampleCount := 0
		// else we call visitTree recursively on the children
		for _, childNode := range node.Children {
			totChildrenSampleCount += childNode.SampleCount
			visitTree(f, childNode, currentStack)
		}

		// If the children's sample count is less than the current
		// nodes sample count, it means there are some samples
		// ending at the current node.
		diff := node.SampleCount - totChildrenSampleCount
		if diff > 0 {
			f.AddStack(*currentStack, diff)
		}
	}
	// pop last element before returning
	*currentStack = (*currentStack)[:len(*currentStack)-1]
}

/*
func updateCounterAndStacks(
	stacks *[][]frame.Frame,
	counter map[uint64]int,
	currentStack *[]frame.Frame,
	fingerprint uint64,
	sampleCount int) {
	if _, exists := counter[fingerprint]; exists {
		counter[fingerprint] += sampleCount
	} else {
		counter[fingerprint] = sampleCount
		cp := make([]frame.Frame, len(*currentStack))
		copy(cp, *currentStack)
		*stacks = append(*stacks, cp)
	}
}*/

// Here we define a function getFrameID instead
// of reusing Frame.ID() method because we might
// want to do things differently and only solve
// at symbol level (ignoring line and instruction_addr)
func getFrameID(f frame.Frame) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s:%s", f.Function, f.Package)))
	return hex.EncodeToString(hash[:])
}

func GetFlamegraphFromProfiles(
	ctx context.Context,
	profilesBucket *storage.BucketHandle,
	organizationID uint64,
	projectID uint64,
	profileIDs []string,
	numWorkers int,
	timeout time.Duration) (speedscope.Output, error) {

	if numWorkers < 1 {
		numWorkers = 1
	}
	var wg sync.WaitGroup
	flamegraph := Flamegraph{counters: make(map[uint64]int)}
	callTreesQueue := make(chan map[uint64][]*nodetree.Node, numWorkers)
	profileIDsChan := make(chan string, numWorkers)
	hub := sentry.GetHubFromContext(ctx)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(
			profIDsChan chan string,
			callTreesQueue chan map[uint64][]*nodetree.Node,
			timeout time.Duration) {

			defer wg.Done()
			// each worker should stop if
			// hitting a timeout
			to := time.NewTimer(timeout)
			defer to.Stop()

		DONE:
			for profileID := range profIDsChan {
				select {
				case <-to.C:
					// if we've hit a timeout, stop fetching
					// new profiles
					break DONE
				default:
					var p profile.Profile
					err := storageutil.UnmarshalCompressed(ctx, profilesBucket, profile.StoragePath(organizationID, projectID, profileID), &p)
					if err != nil && !errors.Is(err, context.DeadlineExceeded) {
						hub.CaptureException(err)
						continue
					}
					callTrees, err := p.CallTrees()
					if err != nil {
						hub.CaptureException(err)
						continue
					}
					callTreesQueue <- callTrees
				}
			}

		}(profileIDsChan, callTreesQueue, timeout)
	}

	go func(profIDsChan chan string, profileIDs []string, timeout time.Duration) {
		// if we hit a timeout and we haven't
		// fetched all the profiles yet, stop.
		to := time.NewTimer(timeout)

		for _, profileID := range profileIDs {
			select {
			case <-to.C:
				close(profIDsChan)
				return
			default:
				profIDsChan <- profileID
			}
		}
		close(profIDsChan)

	}(profileIDsChan, profileIDs, timeout)

	go func(callTreesQueue chan map[uint64][]*nodetree.Node) {
		wg.Wait()
		close(callTreesQueue)
	}(callTreesQueue)

	for callTrees := range callTreesQueue {
		ProcessStacksFromCallTrees(callTrees, &flamegraph)
	}

	if len(flamegraph.stacks) == 0 {
		// early return: no need to call `ConvertStackTracesToFlamegraph`
		return speedscope.Output{}, nil
	} else {
		return ConvertStackTracesToFlamegraph(&flamegraph, 4), nil
	}
}
