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
	"github.com/getsentry/vroom/internal/chunk"
	"github.com/getsentry/vroom/internal/metrics"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/storageutil"
	"github.com/getsentry/vroom/internal/utils"
	"gocloud.dev/blob"
)

type (
	Pair[T, U any] struct {
		First  T
		Second U
	}

	CallTrees map[uint64][]*nodetree.Node

	ChunkMetadata struct {
		ProfilerID    string           `json:"profiler_id"`
		ChunkID       string           `json:"chunk_id"`
		SpanIntervals []utils.Interval `json:"span_intervals,omitempty"`
	}
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
	spans *[][]utils.Interval,
	numWorkers int,
	timeout time.Duration) (speedscope.Output, error) {
	if numWorkers < 1 {
		numWorkers = 1
	}
	var wg sync.WaitGroup
	var flamegraphTree []*nodetree.Node
	callTreesQueue := make(chan Pair[string, CallTrees], numWorkers)
	profileIDsChan := make(chan Pair[string, []utils.Interval], numWorkers)
	hub := sentry.GetHubFromContext(ctx)
	timeoutContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(
			profIDsChan chan Pair[string, []utils.Interval],
			callTreesQueue chan Pair[string, CallTrees],
			ctx context.Context) {
			defer wg.Done()

			for profilePair := range profIDsChan {
				profileID := profilePair.First
				spans := profilePair.Second
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
				if spans != nil {
					// span intervals here contains Unix epoch timestamp (in ns).
					// here we replace their value with the ns elapsed since
					// the profile.timestamps to be consistent with the sample/node
					// 'start' and 'end'
					relativeIntervalsFromAbsoluteTimestamp(&spans, uint64(p.Timestamp().UnixNano()))
					sortedSpans := mergeIntervals(&spans)
					for tid, callTree := range callTrees {
						callTrees[tid] = sliceCallTree(&callTree, &sortedSpans)
					}
				}
				callTreesQueue <- Pair[string, CallTrees]{profileID, callTrees}
			}
		}(profileIDsChan, callTreesQueue, timeoutContext)
	}

	go func(profIDsChan chan Pair[string, []utils.Interval], profileIDs []string, ctx context.Context) {
		for i, profileID := range profileIDs {
			select {
			case <-ctx.Done():
				close(profIDsChan)
				return
			default:
				profilePair := Pair[string, []utils.Interval]{First: profileID, Second: nil}
				if spans != nil {
					profilePair.Second = (*spans)[i]
				}
				profIDsChan <- profilePair
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
			addCallTreeToFlamegraph(&flamegraphTree, callTree, annotateWithProfileID(profileID))
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

func annotateWithProfileID(profileID string) func(n *nodetree.Node) {
	return func(n *nodetree.Node) {
		n.ProfileIDs[profileID] = void
	}
}

func annotateWithProfileExample(example utils.ExampleMetadata) func(n *nodetree.Node) {
	return func(n *nodetree.Node) {
		n.Profiles[example] = void
	}
}

func addCallTreeToFlamegraph(flamegraphTree *[]*nodetree.Node, callTree []*nodetree.Node, annotate func(n *nodetree.Node)) {
	for _, node := range callTree {
		if existingNode := getMatchingNode(flamegraphTree, node); existingNode != nil {
			existingNode.SampleCount += node.SampleCount
			existingNode.DurationNS += node.DurationNS
			addCallTreeToFlamegraph(&existingNode.Children, node.Children, annotate)
			if node.SampleCount > sumNodesSampleCount(node.Children) {
				annotate(existingNode)
			}
		} else {
			*flamegraphTree = append(*flamegraphTree, node)
			// in this case since we append the whole branch
			// we haven't had the chance to add the profile IDs
			// to the right children along the branch yet,
			// therefore we call a utility that walk the branch
			// and does it
			expandCallTreeWithProfileID(node, annotate)
		}
	}
}

func expandCallTreeWithProfileID(node *nodetree.Node, annotate func(n *nodetree.Node)) {
	// leaf frames: we  must add the profileID
	if node.Children == nil {
		annotate(node)
	} else {
		childrenSampleCount := 0
		for _, child := range node.Children {
			childrenSampleCount += child.SampleCount
			expandCallTreeWithProfileID(child, annotate)
		}
		// If the children's sample count is less than the current
		// nodes sample count, it means there are some samples
		// ending at the current node. In this case, this node
		// should also contain the profile ID
		if node.SampleCount > childrenSampleCount {
			annotate(node)
		}
	}
}

type flamegraph struct {
	samples           [][]int
	samplesProfileIDs [][]int
	samplesProfiles   [][]int
	sampleCounts      []uint64
	sampleDurationsNs []uint64
	frames            []speedscope.Frame
	framesIndex       map[string]int
	profilesIDsIndex  map[string]int
	profilesIDs       []string
	profilesIndex     map[utils.ExampleMetadata]int
	profiles          []utils.ExampleMetadata
	endValue          uint64
	minFreq           int
}

func toSpeedscope(trees []*nodetree.Node, minFreq int, projectID uint64) speedscope.Output {
	fd := &flamegraph{
		frames:           make([]speedscope.Frame, 0),
		framesIndex:      make(map[string]int),
		minFreq:          minFreq,
		profilesIDsIndex: make(map[string]int),
		profilesIndex:    make(map[utils.ExampleMetadata]int),
		samples:          make([][]int, 0),
		sampleCounts:     make([]uint64, 0),
	}
	for _, tree := range trees {
		stack := make([]int, 0, profile.MaxStackDepth)
		fd.visitCalltree(tree, &stack)
	}

	aggProfiles := make([]interface{}, 1)
	aggProfiles[0] = speedscope.SampledProfile{
		Samples:           fd.samples,
		SamplesProfiles:   fd.samplesProfileIDs,
		SamplesExamples:   fd.samplesProfiles,
		Weights:           fd.sampleCounts,
		SampleCounts:      fd.sampleCounts,
		SampleDurationsNs: fd.sampleDurationsNs,
		IsMainThread:      true,
		Type:              speedscope.ProfileTypeSampled,
		Unit:              speedscope.ValueUnitCount,
		EndValue:          fd.endValue,
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
			Profiles:   fd.profiles,
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
			IsApplication: node.IsApplication,
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
		f.addSample(
			currentStack,
			uint64(node.SampleCount),
			node.DurationNS,
			node.ProfileIDs,
			node.Profiles,
		)
	} else {
		totChildrenSampleCount := 0
		var totChildrenDuration uint64
		// else we call visitTree recursively on the children
		for _, childNode := range node.Children {
			totChildrenSampleCount += childNode.SampleCount
			totChildrenDuration += childNode.DurationNS
			f.visitCalltree(childNode, currentStack)
		}

		// If the children's sample count is less than the current
		// nodes sample count, it means there are some samples
		// ending at the current node.
		diffCount := node.SampleCount - totChildrenSampleCount
		diffDuration := node.DurationNS - totChildrenDuration
		if diffCount >= f.minFreq {
			f.addSample(
				currentStack,
				uint64(diffCount),
				diffDuration,
				node.ProfileIDs,
				node.Profiles,
			)
		}
	}
	// pop last element before returning
	*currentStack = (*currentStack)[:len(*currentStack)-1]
}

func (f *flamegraph) addSample(
	stack *[]int,
	count uint64,
	duration uint64,
	profileIDs map[string]struct{},
	profiles map[utils.ExampleMetadata]struct{},
) {
	cp := make([]int, len(*stack))
	copy(cp, *stack)
	f.samples = append(f.samples, cp)
	f.sampleCounts = append(f.sampleCounts, count)
	f.sampleDurationsNs = append(f.sampleDurationsNs, duration)
	f.samplesProfileIDs = append(f.samplesProfileIDs, f.getProfileIDsIndices(profileIDs))
	f.samplesProfiles = append(f.samplesProfiles, f.getProfilesIndices(profiles))
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

func (f *flamegraph) getProfilesIndices(profiles map[utils.ExampleMetadata]struct{}) []int {
	indices := make([]int, 0, len(profiles))
	for i := range profiles {
		if idx, ok := f.profilesIndex[i]; ok {
			indices = append(indices, idx)
		} else {
			indices = append(indices, len(f.profiles))
			f.profilesIndex[i] = len(f.profiles)
			f.profiles = append(f.profiles, i)
		}
	}
	return indices
}

func GetFlamegraphFromChunks(
	ctx context.Context,
	organizationID uint64,
	projectID uint64,
	storage *blob.Bucket,
	chunksMetadata []ChunkMetadata,
	jobs chan storageutil.ReadJob) (speedscope.Output, error) {
	hub := sentry.GetHubFromContext(ctx)
	results := make(chan storageutil.ReadJobResult, len(chunksMetadata))
	defer close(results)

	chunkIDToMetadata := make(map[string]ChunkMetadata)
	for _, chunkMetadata := range chunksMetadata {
		chunkIDToMetadata[chunkMetadata.ChunkID] = chunkMetadata
		jobs <- chunk.ReadJob{
			Ctx:            ctx,
			ProfilerID:     chunkMetadata.ProfilerID,
			ChunkID:        chunkMetadata.ChunkID,
			OrganizationID: organizationID,
			ProjectID:      projectID,
			Storage:        storage,
			Result:         results,
		}
	}

	var flamegraphTree []*nodetree.Node
	countChunksAggregated := 0
	// read the output of each tasks
	for i := 0; i < len(chunksMetadata); i++ {
		res := <-results
		result, ok := res.(chunk.ReadJobResult)
		if !ok {
			continue
		}
		if result.Err != nil {
			if errors.Is(result.Err, storageutil.ErrObjectNotFound) {
				continue
			}
			if errors.Is(result.Err, context.DeadlineExceeded) {
				return speedscope.Output{}, result.Err
			}
			if hub != nil {
				hub.CaptureException(result.Err)
			}
			continue
		}
		cm := chunkIDToMetadata[result.Chunk.ID]
		for _, interval := range cm.SpanIntervals {
			callTrees, err := result.Chunk.CallTrees(&interval.ActiveThreadID)
			if err != nil {
				if hub != nil {
					hub.CaptureException(err)
				}
				continue
			}
			intervals := []utils.Interval{interval}

			annotate := annotateWithProfileExample(
				utils.NewExampleFromProfilerChunk(
					result.Chunk.ProjectID,
					result.Chunk.ProfilerID,
					result.Chunk.ID,
					result.TransactionID,
					result.ThreadID,
					result.Start,
					result.End,
				),
			)
			for _, callTree := range callTrees {
				slicedTree := sliceCallTree(&callTree, &intervals)
				addCallTreeToFlamegraph(&flamegraphTree, slicedTree, annotate)
			}
		}
		countChunksAggregated++
	}

	sp := toSpeedscope(flamegraphTree, 4, projectID)
	if hub != nil {
		hub.Scope().SetTag("processed_chunks", strconv.Itoa(countChunksAggregated))
	}
	return sp, nil
}

func GetFlamegraphFromCandidates(
	ctx context.Context,
	storage *blob.Bucket,
	organizationID uint64,
	transactionProfileCandidates []utils.TransactionProfileCandidate,
	continuousProfileCandidates []utils.ContinuousProfileCandidate,
	jobs chan storageutil.ReadJob,
	ma *metrics.Aggregator,
) (speedscope.Output, error) {
	hub := sentry.GetHubFromContext(ctx)

	numCandidates := len(transactionProfileCandidates) + len(continuousProfileCandidates)

	results := make(chan storageutil.ReadJobResult, numCandidates)
	defer close(results)

	for _, candidate := range transactionProfileCandidates {
		jobs <- profile.ReadJob{
			Ctx:            ctx,
			OrganizationID: organizationID,
			ProjectID:      candidate.ProjectID,
			ProfileID:      candidate.ProfileID,
			Storage:        storage,
			Result:         results,
		}
	}

	for _, candidate := range continuousProfileCandidates {
		jobs <- chunk.ReadJob{
			Ctx:            ctx,
			OrganizationID: organizationID,
			ProjectID:      candidate.ProjectID,
			ProfilerID:     candidate.ProfilerID,
			ChunkID:        candidate.ChunkID,
			TransactionID:  candidate.TransactionID,
			ThreadID:       candidate.ThreadID,
			Start:          candidate.Start,
			End:            candidate.End,
			Intervals:      candidate.Intervals,
			Storage:        storage,
			Result:         results,
		}
	}

	var flamegraphTree []*nodetree.Node

	for i := 0; i < numCandidates; i++ {
		res := <-results

		err := res.Error()
		if err != nil {
			if errors.Is(err, storageutil.ErrObjectNotFound) {
				continue
			}
			if errors.Is(err, context.DeadlineExceeded) {
				return speedscope.Output{}, err
			}
			if hub != nil {
				hub.CaptureException(err)
			}
			continue
		}

		if result, ok := res.(profile.ReadJobResult); ok {
			profileCallTrees, err := result.Profile.CallTrees()
			if err != nil {
				hub.CaptureException(err)
				continue
			}

			example := utils.NewExampleFromProfileID(result.Profile.ProjectID(), result.Profile.ID())
			annotate := annotateWithProfileExample(example)

			for _, callTree := range profileCallTrees {
				addCallTreeToFlamegraph(&flamegraphTree, callTree, annotate)
			}
			// if metrics aggregator is not null, while we're at it,
			// compute the metrics as well
			if ma != nil {
				functions := metrics.CapAndFilterFunctions(metrics.ExtractFunctionsFromCallTrees(profileCallTrees), int(ma.MaxUniqueFunctions), true)
				ma.AddFunctions(functions, example)
			}
		} else if result, ok := res.(chunk.ReadJobResult); ok {
			for tid, intervals := range result.Intervals {
				chunkCallTrees, err := result.Chunk.CallTrees(&tid)
				if err != nil {
					hub.CaptureException(err)
					continue
				}
				sortedAndMergedIntervals := mergeIntervals(&intervals)
				example := utils.NewExampleFromProfilerChunk(
					result.Chunk.ProjectID,
					result.Chunk.ProfilerID,
					result.Chunk.ID,
					result.TransactionID,
					&tid,
					sortedAndMergedIntervals[0].Start,
					sortedAndMergedIntervals[len(sortedAndMergedIntervals)-1].End,
				)
				annotate := annotateWithProfileExample(example)

				for _, callTree := range chunkCallTrees {
					if len(sortedAndMergedIntervals) > 0 {
						callTree = sliceCallTree(&callTree, &sortedAndMergedIntervals)
					}
					addCallTreeToFlamegraph(&flamegraphTree, callTree, annotate)
				}
				// if metrics aggregator is not null, while we're at it,
				// compute the metrics as well
				if ma != nil {
					functions := metrics.CapAndFilterFunctions(metrics.ExtractFunctionsFromCallTrees(chunkCallTrees), int(ma.MaxUniqueFunctions), true)
					ma.AddFunctions(functions, example)
				}
			} // end --> for tid, intervals := range result.Intervals
		} else {
			// This should never happen
			return speedscope.Output{}, errors.New("unexpected result from storage")
		}
	} // end --> for i := 0; i < numCandidates; i++ {

	sp := toSpeedscope(flamegraphTree, 4, 0)
	if ma != nil {
		fm := ma.ToMetrics()
		sp.Metrics = &fm
	}
	return sp, nil
}
