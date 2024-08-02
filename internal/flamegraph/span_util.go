package flamegraph

import (
	"math"
	"sort"
	"time"

	"github.com/getsentry/vroom/internal/nodetree"
)

type SpanInterval struct {
	Start          uint64 `json:"start,string"`
	End            uint64 `json:"end,string"`
	ActiveThreadID string `json:"active_thread_id"`
}

func mergeIntervals(intervals *[]SpanInterval) []SpanInterval {
	if len(*intervals) == 0 {
		return *intervals
	}
	sort.SliceStable((*intervals), func(i, j int) bool {
		if (*intervals)[i].Start == (*intervals)[j].Start {
			return (*intervals)[i].End < (*intervals)[j].End
		}
		return (*intervals)[i].Start < (*intervals)[j].Start
	})

	newIntervals := []SpanInterval{(*intervals)[0]}
	for _, interval := range (*intervals)[1:] {
		if interval.Start <= newIntervals[len(newIntervals)-1].End {
			newIntervals[len(newIntervals)-1].End = max(newIntervals[len(newIntervals)-1].End, interval.End)
		} else {
			newIntervals = append(newIntervals, interval)
		}
	}

	return newIntervals
}

func relativeIntervalsFromAbsoluteTimestamp(intervals *[]SpanInterval, t uint64) {
	for i, v := range *intervals {
		// safety check: in case the start/end should be
		// earlier than the profile start
		(*intervals)[i].Start = max(t, v.Start) - t
		(*intervals)[i].End = max(t, v.End) - t
	}
}

func sliceCallTree(callTree *[]*nodetree.Node, intervals *[]SpanInterval) []*nodetree.Node {
	slicedTree := make([]*nodetree.Node, 0)
	for _, node := range *callTree {
		if duration := getTotalOverlappingDuration(node, intervals); duration > 0 {
			sampleCount := int(math.Ceil(float64(duration) / float64(time.Millisecond*10)))
			// here we take the minimum between the node sample count and the estimated
			// sample count to mitigate the case when we make a wrong estimation due
			// to sampling frequency not being respected. (Python native code holding
			// the GIL, php, etc.)
			node.SampleCount = min(sampleCount, node.SampleCount)
			node.DurationNS = duration
			if children := sliceCallTree(&node.Children, intervals); len(children) > 0 {
				node.Children = children
			} else {
				node.Children = nil
			}
			slicedTree = append(slicedTree, node)
		}
	} // end range callTree
	return slicedTree
}

func getTotalOverlappingDuration(node *nodetree.Node, intervals *[]SpanInterval) uint64 {
	var duration uint64
	for _, interval := range *intervals {
		if node.EndNS <= interval.Start {
			// in this case any remaining interval
			// starts after the given call frame
			// therefeore we can bail out early
			break
		}
		duration += overlappingDuration(node, &interval)
	}
	return duration
}

func overlappingDuration(node *nodetree.Node, interval *SpanInterval) uint64 {
	end := min(node.EndNS, interval.End)
	start := max(node.StartNS, interval.Start)

	if end <= start {
		return 0
	}
	return end - start
}
