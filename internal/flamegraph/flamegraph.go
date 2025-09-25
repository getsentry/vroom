package flamegraph

import (
	"container/heap"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"gocloud.dev/blob"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/chunk"
	"github.com/getsentry/vroom/internal/examples"
	"github.com/getsentry/vroom/internal/metrics"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/storageutil"
)

type (
	Pair[T, U any] struct {
		First  T
		Second U
	}

	CallTrees map[uint64][]*nodetree.Node
)

var void = struct{}{}

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

func annotateWithProfileExample(example examples.ExampleMetadata) func(n *nodetree.Node) {
	return func(n *nodetree.Node) {
		n.Profiles[example] = void
	}
}

func addCallTreeToFlamegraph(flamegraphTree *[]*nodetree.Node, callTree []*nodetree.Node, annotate func(n *nodetree.Node)) {
	for _, node := range callTree {
		var currentNode *nodetree.Node
		if existingNode := getMatchingNode(flamegraphTree, node); existingNode != nil {
			currentNode = existingNode
			currentNode.Occurrence += node.Occurrence
			currentNode.SampleCount += node.SampleCount
			currentNode.DurationNS += node.DurationNS
			currentNode.SelfTimeNS += node.SelfTimeNS
			currentNode.DurationsNS = append(currentNode.DurationsNS, node.DurationsNS...)
		} else {
			currentNode = node.ShallowCopyWithoutChildren()
			*flamegraphTree = append(*flamegraphTree, currentNode)
		}
		addCallTreeToFlamegraph(&currentNode.Children, node.Children, annotate)
		if node.SampleCount > sumNodesSampleCount(node.Children) {
			annotate(currentNode)
		}
	}
}

type (
	flamegraph struct {
		samples           [][]int
		samplesProfiles   [][]int
		sampleCounts      []uint64
		sampleDurationsNs []uint64
		frames            []speedscope.Frame
		framesIndex       map[string]int
		frameInfos        []speedscope.FrameInfo
		profilesIndex     map[examples.ExampleMetadata]int
		profiles          []examples.ExampleMetadata
		endValue          uint64
		maxSamples        int
		// The total number of samples that were added to the flamegraph
		// including the ones that were dropped due to them exceeding
		// the max samples limit.
		totalSamples int
	}

	flamegraphSample struct {
		stack    []int
		count    uint64 // count refers to the individual sample counts
		duration uint64
		profiles map[examples.ExampleMetadata]struct{}
	}
)

func (f *flamegraph) overCapacity() bool {
	return f.Len() > f.maxSamples
}

func (f *flamegraph) Len() int {
	// assumes all the sample* slices have the same length
	return len(f.samples)
}

func (f *flamegraph) Less(i, j int) bool {
	// first compare the counts per sample
	if f.sampleCounts[i] != f.sampleCounts[j] {
		return f.sampleCounts[i] < f.sampleCounts[j]
	}
	// if counts are equal, compare the duration per sample
	if f.sampleDurationsNs[i] != f.sampleDurationsNs[j] {
		return f.sampleDurationsNs[i] < f.sampleDurationsNs[j]
	}
	// if durations are equal, compare the depth per sample
	return len(f.samples[i]) < len(f.samples[j])
}

func (f *flamegraph) Swap(i, j int) {
	f.samples[i], f.samples[j] = f.samples[j], f.samples[i]
	f.samplesProfiles[i], f.samplesProfiles[j] = f.samplesProfiles[j], f.samplesProfiles[i]
	f.sampleCounts[i], f.sampleCounts[j] = f.sampleCounts[j], f.sampleCounts[i]
	f.sampleDurationsNs[i], f.sampleDurationsNs[j] = f.sampleDurationsNs[j], f.sampleDurationsNs[i]
}

func (f *flamegraph) Push(item any) {
	sample := item.(flamegraphSample)

	f.samples = append(f.samples, sample.stack)
	f.sampleCounts = append(f.sampleCounts, sample.count)
	f.sampleDurationsNs = append(f.sampleDurationsNs, sample.duration)
	f.samplesProfiles = append(f.samplesProfiles, f.getProfilesIndices(sample.profiles))
}

func (f *flamegraph) Pop() any {
	n := len(f.samples) - 1

	profiles := make(map[examples.ExampleMetadata]struct{})
	for _, i := range f.samplesProfiles[n] {
		profiles[f.profiles[i]] = struct{}{}
	}

	sample := flamegraphSample{
		stack:    f.samples[n],
		count:    f.sampleCounts[n],
		duration: f.sampleDurationsNs[n],
		profiles: profiles,
	}

	f.samples = f.samples[0:n]
	f.sampleCounts = f.sampleCounts[0:n]
	f.sampleDurationsNs = f.sampleDurationsNs[0:n]
	f.samplesProfiles = f.samplesProfiles[0:n]

	return sample
}

func toSpeedscope(
	ctx context.Context,
	trees []*nodetree.Node,
	maxSamples int,
	projectID uint64,
) speedscope.Output {
	s := sentry.StartSpan(ctx, "processing")
	s.Description = "generating speedscope"
	defer s.Finish()

	fd := &flamegraph{
		frames:        make([]speedscope.Frame, 0),
		frameInfos:    make([]speedscope.FrameInfo, 0),
		framesIndex:   make(map[string]int),
		maxSamples:    maxSamples,
		profilesIndex: make(map[examples.ExampleMetadata]int),
		samples:       make([][]int, 0),
		sampleCounts:  make([]uint64, 0),
	}
	for _, tree := range trees {
		stack := make([]int, 0, profile.MaxStackDepth)
		fd.visitCalltree(tree, &stack)
	}

	for i, frameInfo := range fd.frameInfos {
		sort.Slice(frameInfo.DurationsNS, func(i, j int) bool {
			return frameInfo.DurationsNS[i] < frameInfo.DurationsNS[j]
		})
		frameInfo.P75Duration, _ = metrics.Quantile(frameInfo.DurationsNS, 0.75)
		frameInfo.P95Duration, _ = metrics.Quantile(frameInfo.DurationsNS, 0.95)
		frameInfo.P99Duration, _ = metrics.Quantile(frameInfo.DurationsNS, 0.99)
		fd.frameInfos[i] = frameInfo
	}

	s.SetData("total_samples", fd.totalSamples)
	s.SetData("final_samples", fd.Len())

	aggProfiles := make([]interface{}, 1)
	aggProfiles[0] = speedscope.SampledProfile{
		Samples:           fd.samples,
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
			FrameInfos: fd.frameInfos,
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
	frameID := getIDFromNode(node)
	if i, exists := f.framesIndex[frameID]; exists {
		*currentStack = append(*currentStack, i)
		f.frameInfos[i].Count += node.Occurrence
		f.frameInfos[i].Weight += node.DurationNS
		f.frameInfos[i].SumDuration += node.DurationNS
		f.frameInfos[i].SumSelfTime += node.SelfTimeNS
		f.frameInfos[i].DurationsNS = append(f.frameInfos[i].DurationsNS, node.DurationsNS...)
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
			Fingerprint:   frame.Fingerprint(),
		}
		f.framesIndex[frameID] = len(f.frames)
		*currentStack = append(*currentStack, len(f.frames))
		f.frames = append(f.frames, sfr)
		f.frameInfos = append(f.frameInfos, speedscope.FrameInfo{
			Count:       node.Occurrence,
			Weight:      node.DurationNS,
			SumDuration: node.DurationNS,
			SumSelfTime: node.SelfTimeNS,
			DurationsNS: node.DurationsNS,
		})
	}

	// base case (when we reach leaf frames)
	if node.Children == nil {
		f.addSample(
			currentStack,
			uint64(node.SampleCount),
			node.DurationNS,
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
		if diffCount > 0 {
			f.addSample(
				currentStack,
				uint64(diffCount),
				diffDuration,
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
	profiles map[examples.ExampleMetadata]struct{},
) {
	f.totalSamples++
	cp := make([]int, len(*stack))
	copy(cp, *stack)

	heap.Push(f, flamegraphSample{
		stack:    cp,
		count:    count,
		duration: duration,
		profiles: profiles,
	})
	for f.overCapacity() {
		heap.Pop(f)
	}
	f.endValue += count
}

func (f *flamegraph) getProfilesIndices(profilesMap map[examples.ExampleMetadata]struct{}) []int {
	profiles := make([]examples.ExampleMetadata, 0, len(profilesMap))
	for profile := range profilesMap {
		profiles = append(profiles, profile)
	}
	sort.Slice(profiles, func(i int, j int) bool {
		profile1 := profiles[i]
		profile2 := profiles[j]

		if profile1.ProfileID != "" {
			return profile1.ProfileID < profile2.ProfileID
		}

		if profile2.ProfileID != "" {
			return true
		}

		return profile1.ProfilerID < profile2.ProfilerID
	})

	indices := make([]int, 0, len(profiles))
	for _, i := range profiles {
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

func GetFlamegraphFromCandidates(
	ctx context.Context,
	storage *blob.Bucket,
	organizationID uint64,
	transactionProfileCandidates []examples.TransactionProfileCandidate,
	continuousProfileCandidates []examples.ContinuousProfileCandidate,
	jobs chan storageutil.ReadJob,
	ma *metrics.Aggregator,
	span *sentry.Span,
) (speedscope.Output, error) {
	hub := sentry.GetHubFromContext(ctx)

	results := make(chan storageutil.ReadJobResult)
	defer close(results)

	go func() {
		dispatchSpan := span.StartChild("dispatch candidates")
		dispatchSpan.SetData("transaction_candidates", len(transactionProfileCandidates))
		dispatchSpan.SetData("continuous_candidates", len(continuousProfileCandidates))

		for _, candidate := range transactionProfileCandidates {
			jobs <- profile.CallTreesReadJob{
				Ctx:            ctx,
				OrganizationID: organizationID,
				ProjectID:      candidate.ProjectID,
				ProfileID:      candidate.ProfileID,
				Storage:        storage,
				Result:         results,
			}
		}

		for _, candidate := range continuousProfileCandidates {
			jobs <- chunk.CallTreesReadJob{
				Ctx:            ctx,
				OrganizationID: organizationID,
				ProjectID:      candidate.ProjectID,
				ProfilerID:     candidate.ProfilerID,
				ChunkID:        candidate.ChunkID,
				TransactionID:  candidate.TransactionID,
				ThreadID:       candidate.ThreadID,
				Start:          candidate.Start,
				End:            candidate.End,
				Storage:        storage,
				Result:         results,
			}
		}

		dispatchSpan.Finish()
	}()

	var flamegraphTree []*nodetree.Node

	flamegraphSpan := span.StartChild("processing candidates")

	numCandidates := len(transactionProfileCandidates) + len(continuousProfileCandidates)

	for i := 0; i < numCandidates; i++ {
		res := <-results

		err := res.Error()
		if err != nil {
			if errors.Is(err, storageutil.ErrObjectNotFound) {
				continue
			}
			if errors.Is(err, context.DeadlineExceeded) {
				// Since we set an artificially lower timeout
				// (10s < 15s), if we exceeded the deadline
				// we stopped downloading chunks, but we
				// still have time to compute the flamegraph
				// with the chunks we downloaded so far
				// and return it.
				continue
			}
			if hub != nil {
				hub.CaptureException(err)
			}
			continue
		}

		if result, ok := res.(profile.CallTreesReadJobResult); ok {
			transactionProfileSpan := span.StartChild("calltree")
			transactionProfileSpan.Description = "transaction profile"

			start, end := result.Profile.StartAndEndEpoch()
			example := examples.NewExampleFromProfileID(
				result.Profile.ProjectID(),
				result.Profile.ID(),
				start,
				end,
			)
			annotate := annotateWithProfileExample(example)

			for _, callTree := range result.CallTrees {
				addCallTreeToFlamegraph(&flamegraphTree, callTree, annotate)
			}
			// if metrics aggregator is not null, while we're at it,
			// compute the metrics as well
			if ma != nil {
				functions := metrics.CapAndFilterFunctions(metrics.ExtractFunctionsFromCallTrees(result.CallTrees, ma.MinDepth), int(ma.MaxUniqueFunctions), true)
				ma.AddFunctions(functions, example)
			}

			transactionProfileSpan.Finish()
		} else if result, ok := res.(chunk.CallTreesReadJobResult); ok {
			chunkProfileSpan := span.StartChild("calltree")
			chunkProfileSpan.Description = "continuous profile"

			for threadID, callTree := range result.CallTrees {
				if result.Start > 0 && result.End > 0 {
					interval := examples.Interval{
						Start: result.Start,
						End:   result.End,
					}
					callTree = sliceCallTree(&callTree, &[]examples.Interval{interval})
				}

				example := examples.NewExampleFromProfilerChunk(
					result.Chunk.GetProjectID(),
					result.Chunk.GetProfilerID(),
					result.Chunk.GetID(),
					result.TransactionID,
					&threadID,
					result.Start,
					result.End,
				)
				annotate := annotateWithProfileExample(example)

				addCallTreeToFlamegraph(&flamegraphTree, callTree, annotate)

				// if metrics aggregator is not null, while we're at it,
				// compute the metrics as well
				if ma != nil {
					functions := metrics.CapAndFilterFunctions(metrics.ExtractFunctionsFromCallTreesForThread(callTree, ma.MinDepth), int(ma.MaxUniqueFunctions), true)
					ma.AddFunctions(functions, example)
				}
			}
			chunkProfileSpan.Finish()
		} else {
			// This should never happen
			return speedscope.Output{}, errors.New("unexpected result from storage")
		}
	}

	flamegraphSpan.Finish()

	serializeSpan := span.StartChild("serialize")
	defer serializeSpan.Finish()

	sp := toSpeedscope(ctx, flamegraphTree, 1000, 0)
	if ma != nil {
		fm := ma.ToMetrics()
		sp.Metrics = &fm
	}
	return sp, nil
}
