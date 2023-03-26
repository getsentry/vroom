package flamegraph

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/storageutil"
	"gocloud.dev/blob"
)

type (
	Pair[T, U any] struct {
		First  T
		Second U
	}

	CallTrees map[uint64][]*nodetree.Node
)

var (
	void = struct{}{}
)

func GetFlamegraphFromProfiles(
	ctx context.Context,
	profilesBucket *blob.Bucket,
	organizationID uint64,
	projectID uint64,
	profileIDs []string,
	numWorkers int,
	timeout time.Duration) (speedscope.Output, error) {
	if numWorkers < 1 {
		numWorkers = 1
	}
	var wg sync.WaitGroup
	var flamegraphTree []*nodetree.Node
	callTreesQueue := make(chan Pair[string, CallTrees], numWorkers)
	profileIDsChan := make(chan string, numWorkers)
	hub := sentry.GetHubFromContext(ctx)
	timeoutContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(
			profIDsChan chan string,
			callTreesQueue chan Pair[string, CallTrees],
			ctx context.Context) {
			defer wg.Done()

			for profileID := range profIDsChan {
				var p profile.Profile
				err := storageutil.UnmarshalCompressed(ctx, profilesBucket, profile.StoragePath(organizationID, projectID, profileID), &p)
				if err != nil {
					if errors.Is(err, storageutil.ErrObjectNotFound) {
						continue
					}
					if errors.Is(err, context.DeadlineExceeded) {
						return
					}
					hub.CaptureException(err)
					continue
				}
				callTrees, err := p.CallTrees()
				if err != nil {
					hub.CaptureException(err)
					continue
				}
				callTreesQueue <- Pair[string, CallTrees]{profileID, callTrees}
			}
		}(profileIDsChan, callTreesQueue, timeoutContext)
	}

	go func(profIDsChan chan string, profileIDs []string, ctx context.Context) {
		for _, profileID := range profileIDs {
			select {
			case <-ctx.Done():
				close(profIDsChan)
				return
			default:
				profIDsChan <- profileID
			}
		}
		close(profIDsChan)
	}(profileIDsChan, profileIDs, timeoutContext)

	go func(callTreesQueue chan Pair[string, CallTrees]) {
		wg.Wait()
		close(callTreesQueue)
	}(callTreesQueue)

	countProfAggregated := 0
	for pair := range callTreesQueue {
		profileID := pair.First
		for _, callTree := range pair.Second {
			addCallTreeToFlamegraph(&flamegraphTree, callTree, profileID)
		}
		countProfAggregated++
	}

	sp := toSpeedscope(flamegraphTree, 4, projectID)
	hub.Scope().SetTag("processed_profiles", strconv.Itoa(countProfAggregated))
	return sp, nil
}

func getMatchingNode(nodes *[]*nodetree.Node, newNode *nodetree.Node) *nodetree.Node {
	for _, node := range *nodes {
		if node.Name == newNode.Name && node.Package == newNode.Package {
			return node
		}
	}
	return nil
}

func sumNodesSampleCount(nodes []*nodetree.Node) int {
	c := 0
	for _, node := range nodes {
		c += node.SampleCount
	}
	return c
}

func addCallTreeToFlamegraph(flamegraphTree *[]*nodetree.Node, callTree []*nodetree.Node, profileID string) {
	for _, node := range callTree {
		if existingNode := getMatchingNode(flamegraphTree, node); existingNode != nil {
			existingNode.SampleCount += node.SampleCount
			if node.SampleCount > sumNodesSampleCount(node.Children) {
				existingNode.ProfileIDs[profileID] = void
			}
			addCallTreeToFlamegraph(&existingNode.Children, node.Children, profileID)
		} else {
			*flamegraphTree = append(*flamegraphTree, node)
			// in this case since we append the whole branch
			// we haven't had the chance to add the profile IDs
			// to the right children along the branch yet,
			// therefore we call a utility that walk the branch
			// and does it
			expandCallTreeWithProfileID(node, profileID)
		}
	}
}

func expandCallTreeWithProfileID(node *nodetree.Node, profileID string) {
	// leaf frames: we  must add the profileID
	if node.Children == nil {
		node.ProfileIDs[profileID] = void
	} else {
		childrenSampleCount := 0
		for _, child := range node.Children {
			childrenSampleCount += child.SampleCount
			expandCallTreeWithProfileID(child, profileID)
		}
		// If the children's sample count is less than the current
		// nodes sample count, it means there are some samples
		// ending at the current node. In this case, this node
		// should also contain the profile ID
		if node.SampleCount > childrenSampleCount {
			node.ProfileIDs[profileID] = void
		}
	}
}

type flamegraph struct {
	samples           [][]int
	samplesProfileIDs [][]int
	weights           []uint64
	frames            []speedscope.Frame
	framesIndex       map[string]int
	profilesIDsIndex  map[string]int
	profilesIDs       []string
	endValue          uint64
	minFreq           int
}

func toSpeedscope(trees []*nodetree.Node, minFreq int, projectID uint64) speedscope.Output {
	fd := &flamegraph{
		frames:           make([]speedscope.Frame, 0),
		framesIndex:      make(map[string]int),
		minFreq:          minFreq,
		profilesIDsIndex: make(map[string]int),
		samples:          make([][]int, 0),
		weights:          make([]uint64, 0),
	}
	for _, tree := range trees {
		stack := make([]int, 0, 128)
		fd.visitCalltree(tree, &stack)
	}

	aggProfiles := make([]interface{}, 1)
	aggProfiles[0] = speedscope.SampledProfile{
		Samples:         fd.samples,
		SamplesProfiles: fd.samplesProfileIDs,
		Weights:         fd.weights,
		IsMainThread:    true,
		Type:            speedscope.ProfileTypeSampled,
		Unit:            speedscope.ValueUnitCount,
		EndValue:        fd.endValue,
	}

	return speedscope.Output{
		Metadata: speedscope.ProfileMetadata{
			ProfileView: speedscope.ProfileView{
				ProjectID: projectID,
			},
		},
		Shared: speedscope.SharedData{
			Frames:     fd.frames,
			ProfileIDs: fd.profilesIDs,
		},
		Profiles: aggProfiles,
	}
}

func getIDFromNode(node *nodetree.Node) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s:%s", node.Name, node.Package)))
	return hex.EncodeToString(hash[:])
}

func (f *flamegraph) visitCalltree(node *nodetree.Node, currentStack *[]int) {
	if node.SampleCount < f.minFreq {
		return
	}

	frameID := getIDFromNode(node)
	if i, exists := f.framesIndex[frameID]; exists {
		*currentStack = append(*currentStack, i)
	} else {
		frame := node.ToFrame()
		sfr := speedscope.Frame{
			Name:          frame.Function,
			Image:         frame.ModuleOrPackage(),
			Path:          frame.Path,
			IsApplication: *frame.InApp,
			Col:           frame.Column,
			File:          frame.File,
			Inline:        frame.IsInline(),
			Line:          frame.Line,
		}
		f.framesIndex[frameID] = len(f.frames)
		*currentStack = append(*currentStack, len(f.frames))
		f.frames = append(f.frames, sfr)
	}

	// base case (when we reach leaf frames)
	if node.Children == nil {
		f.addSample(currentStack, uint64(node.SampleCount), node.ProfileIDs)
	} else {
		totChildrenSampleCount := 0
		// else we call visitTree recursively on the children
		for _, childNode := range node.Children {
			totChildrenSampleCount += childNode.SampleCount
			f.visitCalltree(childNode, currentStack)
		}

		// If the children's sample count is less than the current
		// nodes sample count, it means there are some samples
		// ending at the current node.
		diff := node.SampleCount - totChildrenSampleCount
		if diff >= f.minFreq {
			f.addSample(currentStack, uint64(diff), node.ProfileIDs)
		}
	}
	// pop last element before returning
	*currentStack = (*currentStack)[:len(*currentStack)-1]
}

func (f *flamegraph) addSample(stack *[]int, count uint64, profileIDs map[string]struct{}) {
	cp := make([]int, len(*stack))
	copy(cp, *stack)
	f.samples = append(f.samples, cp)
	f.weights = append(f.weights, count)
	f.samplesProfileIDs = append(f.samplesProfileIDs, f.getProfileIDsIndices(profileIDs))
	f.endValue += count
}

func (f *flamegraph) getProfileIDsIndices(profileIDs map[string]struct{}) []int {
	indices := make([]int, 0, len(profileIDs))
	for id := range profileIDs {
		if idx, ok := f.profilesIDsIndex[id]; ok {
			indices = append(indices, idx)
		} else {
			indices = append(indices, len(f.profilesIDs))
			f.profilesIDsIndex[id] = len(f.profilesIDs)
			f.profilesIDs = append(f.profilesIDs, id)
		}
	}
	return indices
}
