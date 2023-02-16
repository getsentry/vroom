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
	stacks      [][]int
	weights     []uint64
	frames      []speedscope.Frame
	framesIndex map[uint64]int
	stackIndex  map[uint64]int
}

func (f *Flamegraph) AddStack(stack []frame.Frame, count uint64) {
	fingerprint := stack[len(stack)-1].Fingerprint
	if _, exists := f.stackIndex[fingerprint]; exists {
		f.weights[f.stackIndex[fingerprint]] += count
	} else {
		st := make([]int, len(stack))
		for i, frame := range stack {
			if _, exists := f.framesIndex[fingerprint]; exists {
				st[i] = f.framesIndex[fingerprint]
			} else {
				fr := speedscope.Frame{
					Name:  frame.Function,
					Image: frame.PackageBaseName(),
					Path:  frame.Path,
				}
				f.framesIndex[frame.Fingerprint] = len(f.frames)
				f.frames = append(f.frames, fr)
				st[i] = len(f.frames) - 1
			}
		}
		f.stackIndex[fingerprint] = len(f.stackIndex)
		f.weights = append(f.weights, count)
		f.stacks = append(f.stacks, st)
	}
}

func (f *Flamegraph) toSpeedscope(minFreq uint64) speedscope.Output {

	// filter out stack traces with a frequency less
	// than minFreq
	n := 0
	for i, stack := range f.stacks {
		if f.weights[i] >= minFreq {
			f.weights[n] = f.weights[i]
			f.stacks[n] = stack
			n++
		}
	}
	f.stacks = f.stacks[:n]
	f.weights = f.weights[:n]

	fmt.Printf("Called -> stacks: %v", len(f.stacks))

	var uniqueFrames []speedscope.Frame
	frameIndex := make(map[string]int)
	var endValue uint64 = 0

	// since we've filtered some stacks, we might have to
	// exclude some frames that are unused.
	// this means stack index
	for i, stack := range f.stacks {
		weight := f.weights[i]
		endValue += uint64(weight)
		for i, index := range stack {
			fr := f.frames[f.framesIndex[uint64(index)]]
			frameID := getFrameID(fr)
			if _, exists := frameIndex[frameID]; exists {
				stack[i] = frameIndex[frameID]
			} else {
				stack[i] = len(uniqueFrames)
				uniqueFrames = append(uniqueFrames, fr)
			}
		}
	}

	aggProfiles := make([]interface{}, 1)
	aggProfiles[0] = speedscope.SampledProfile{
		Samples:      f.stacks,
		Weights:      f.weights,
		IsMainThread: true,
		Type:         speedscope.ProfileTypeSampled,
		Unit:         speedscope.ValueUnitCount,
		EndValue:     endValue,
	}

	return speedscope.Output{
		Shared: speedscope.SharedData{
			Frames: uniqueFrames,
		},
		Profiles: aggProfiles,
	}
}

func ProcessStacksFromCallTrees(callTrees map[uint64][]*nodetree.Node, f *Flamegraph) {

	for _, threadTrees := range callTrees {
		for _, tree := range threadTrees {
			// 128 is the max stack size
			currentStack := make([]frame.Frame, 0, 128)
			AddCalltree(f, tree, &currentStack)
		}
	}
}

func AddCalltree(f *Flamegraph, node *nodetree.Node, currentStack *[]frame.Frame) {
	currentFrame := node.Frame()
	*currentStack = append(*currentStack, currentFrame)

	// base case (when we reach leaf frames)
	if node.Children == nil {
		f.AddStack(*currentStack, uint64(node.SampleCount))
	} else {
		totChildrenSampleCount := 0
		// else we call visitTree recursively on the children
		for _, childNode := range node.Children {
			totChildrenSampleCount += childNode.SampleCount
			AddCalltree(f, childNode, currentStack)
		}

		// If the children's sample count is less than the current
		// nodes sample count, it means there are some samples
		// ending at the current node.
		diff := node.SampleCount - totChildrenSampleCount
		if diff > 0 {
			f.AddStack(*currentStack, uint64(diff))
		}
	}
	// pop last element before returning
	*currentStack = (*currentStack)[:len(*currentStack)-1]
}

// Here we define a function getFrameID instead
// of reusing Frame.ID() method because we might
// want to do things differently and only solve
// at symbol level (ignoring line and instruction_addr)
func getFrameID(f speedscope.Frame) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s:%s", f.Name, f.Image)))
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
	flamegraph := Flamegraph{
		framesIndex: make(map[uint64]int),
		stackIndex:  make(map[uint64]int),
	}
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
		return flamegraph.toSpeedscope(4), nil
	}
}
