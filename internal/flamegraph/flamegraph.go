package flamegraph

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/storageutil"
)

func ConvertStackTracesToFlamegraph(
	stacks *[][]frame.Frame,
	stacksCount map[uint64]int,
	minFreq int) speedscope.Output {

	// filter out stack traces with a frequency less
	// than minFreq
	n := 0
	for _, stack := range *stacks {
		if stacksCount[stack[len(stack)-1].Fingerprint] >= minFreq {
			(*stacks)[n] = stack
			n++
		}
	}
	*stacks = (*stacks)[:n]

	frames := make([]speedscope.Frame, 0)
	samples := make([][]int, 0, len(*stacks))
	addressToIndex := make(map[string]int)
	weights := make([]uint64, 0, len(*stacks))

	for _, stack := range *stacks {
		weight := stacksCount[stack[len(stack)-1].Fingerprint]
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
	}

	return speedscope.Output{
		Shared: speedscope.SharedData{
			Frames: frames,
		},
		Profiles: aggProfiles,
	}
}

func ProcessStacksFromCallTrees(
	callTrees map[uint64][]*nodetree.Node,
	stacks *[][]frame.Frame,
	stacksCount map[uint64]int) {

	for _, threadTrees := range callTrees {
		for _, tree := range threadTrees {
			currentStack := make([]frame.Frame, 0)
			visitTree(stacks, stacksCount, tree, &currentStack)
		}
	}
}

func visitTree(stacks *[][]frame.Frame, counter map[uint64]int, node *nodetree.Node, currentStack *[]frame.Frame) {
	currentFrame := node.Frame()
	*currentStack = append(*currentStack, currentFrame)

	// base case (when we reach leaf frames)
	if node.Children == nil {
		updateCounterAndStacks(stacks, counter, currentStack, node.Fingerprint, node.SampleCount)
		// pop last element before returning
		*currentStack = (*currentStack)[:len(*currentStack)-1]
	} else {
		totChildrenSampleCount := 0
		// else we call visitTree recursively on the children
		for _, childNode := range node.Children {
			totChildrenSampleCount += childNode.SampleCount
			visitTree(stacks, counter, childNode, currentStack)
		}
		// once the children are visited, if node.SampleCount
		// is bigger than totChildrenSampleCount, then it means
		// the current non-leaf node was also the last frame of
		// an independent sampled stack trace.
		// node.SampleCount - totChildrenSampleCount will give us
		// the count for that stack trace
		diff := node.SampleCount - totChildrenSampleCount
		if diff > 0 {
			updateCounterAndStacks(stacks, counter, currentStack, node.Fingerprint, diff)
		}
		// pop last element before returning
		*currentStack = (*currentStack)[:len(*currentStack)-1]
	}
}

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
}

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
	profileIDs []string) (speedscope.Output, error) {

	var stacks [][]frame.Frame
	stacksCount := make(map[uint64]int)

	for _, profileID := range profileIDs {
		var p profile.Profile
		err := storageutil.UnmarshalCompressed(ctx, profilesBucket, profile.StoragePath(organizationID, projectID, profileID), &p)
		if err != nil {
			return speedscope.Output{}, err
		}
		callTrees, err := p.CallTrees()
		if err != nil {
			return speedscope.Output{}, err
		}
		ProcessStacksFromCallTrees(callTrees, &stacks, stacksCount)
	} // end profiles processing

	if len(stacks) == 0 {
		// early return: no need to call `ConvertStackTracesToFlamegraph`
		return speedscope.Output{}, nil
	} else {
		return ConvertStackTracesToFlamegraph(&stacks, stacksCount, 4), nil
	}
}
