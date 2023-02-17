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
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/storageutil"
)

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
	var flamegraphTree []*nodetree.Node
	callTreesQueue := make(chan map[uint64][]*nodetree.Node, numWorkers)
	profileIDsChan := make(chan string, numWorkers)
	hub := sentry.GetHubFromContext(ctx)
	timeoutContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(
			profIDsChan chan string,
			callTreesQueue chan map[uint64][]*nodetree.Node,
			ctx context.Context) {

			defer wg.Done()

			for profileID := range profIDsChan {
				var p profile.Profile
				err := storageutil.UnmarshalCompressed(ctx, profilesBucket, profile.StoragePath(organizationID, projectID, profileID), &p)
				if err != nil {
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
				callTreesQueue <- callTrees
			}

		}(profileIDsChan, callTreesQueue, timeoutContext)
	}

	go func(profIDsChan chan string, profileIDs []string, ctx context.Context) {
		for _, profileID := range profileIDs {
			select {
			case <-timeoutContext.Done():
				close(profIDsChan)
				return
			default:
				profIDsChan <- profileID
			}
		}
		close(profIDsChan)

	}(profileIDsChan, profileIDs, timeoutContext)

	go func(callTreesQueue chan map[uint64][]*nodetree.Node) {
		wg.Wait()
		close(callTreesQueue)
	}(callTreesQueue)

	countProfAggregated := 0
	for callTrees := range callTreesQueue {
		for _, callTree := range callTrees {
			addCallTreeToFlamegraph(&flamegraphTree, callTree)
		}
		countProfAggregated += 1
	}

	sp := toSpeedscope(flamegraphTree, 4)
	sp.CountProcessed = countProfAggregated
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

func addCallTreeToFlamegraph(flamegraphTree *[]*nodetree.Node, callTree []*nodetree.Node) {
	for _, node := range callTree {
		if existingNode := getMatchingNode(flamegraphTree, node); existingNode != nil {
			existingNode.SampleCount += node.SampleCount
			addCallTreeToFlamegraph(&existingNode.Children, node.Children)
		} else {
			*flamegraphTree = append(*flamegraphTree, node)
		}
	}
}

type Flamegraph struct {
	samples     [][]int
	weights     []uint64
	frames      []speedscope.Frame
	framesIndex map[string]int
	endValue    uint64
}

func toSpeedscope(trees []*nodetree.Node, minFreq int) speedscope.Output {
	fd := &Flamegraph{
		framesIndex: make(map[string]int),
	}
	for _, tree := range trees {
		stack := make([]int, 0, 128)
		fd.visitCalltree(tree, &stack, minFreq)
	}

	aggProfiles := make([]interface{}, 1)
	aggProfiles[0] = speedscope.SampledProfile{
		Samples:      fd.samples,
		Weights:      fd.weights,
		IsMainThread: true,
		Type:         speedscope.ProfileTypeSampled,
		Unit:         speedscope.ValueUnitCount,
		EndValue:     fd.endValue,
	}

	return speedscope.Output{
		Shared: speedscope.SharedData{
			Frames: fd.frames,
		},
		Profiles: aggProfiles,
	}
}

func getIDFromNode(node *nodetree.Node) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s:%s", node.Name, node.Package)))
	return hex.EncodeToString(hash[:])
}

func (f *Flamegraph) visitCalltree(node *nodetree.Node, currentStack *[]int, minFreq int) {
	if node.SampleCount < minFreq {
		return
	}

	frameID := getIDFromNode(node)
	if i, exists := f.framesIndex[frameID]; exists {
		*currentStack = append(*currentStack, i)
	} else {
		fr := node.Frame()
		sfr := speedscope.Frame{
			Name:  fr.Function,
			Image: fr.PackageBaseName(),
			Path:  fr.Path,
		}
		f.framesIndex[frameID] = len(f.frames)
		*currentStack = append(*currentStack, len(f.frames))
		f.frames = append(f.frames, sfr)
	}

	// base case (when we reach leaf frames)
	if node.Children == nil {
		f.addSample(currentStack, uint64(node.SampleCount))
	} else {
		totChildrenSampleCount := 0
		// else we call visitTree recursively on the children
		for _, childNode := range node.Children {
			totChildrenSampleCount += childNode.SampleCount
			f.visitCalltree(childNode, currentStack, minFreq)
		}

		// If the children's sample count is less than the current
		// nodes sample count, it means there are some samples
		// ending at the current node.
		diff := node.SampleCount - totChildrenSampleCount
		if diff >= minFreq {
			f.addSample(currentStack, uint64(diff))
		}
	}
	// pop last element before returning
	*currentStack = (*currentStack)[:len(*currentStack)-1]

}

func (f *Flamegraph) addSample(stack *[]int, count uint64) {
	cp := make([]int, len(*stack))
	copy(cp, *stack)
	f.samples = append(f.samples, cp)
	f.weights = append(f.weights, count)
	f.endValue += count
}
