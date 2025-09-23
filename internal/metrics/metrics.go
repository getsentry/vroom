package metrics

import (
	"container/heap"
	"errors"
	"math"
	"sort"
	"strconv"

	"github.com/getsentry/vroom/internal/examples"
	"github.com/getsentry/vroom/internal/nodetree"
)

type (
	FunctionsMetadata struct {
		MaxVal   uint64
		Worst    examples.ExampleMetadata
		Examples []examples.ExampleMetadata
	}

	Aggregator struct {
		callTreeFunctionsMap map[uint32]*FunctionMetric
		callTreeFunctions    []*FunctionMetric
		maxUniqueFunctions   int
		maxNumberOfExamples  int
		// For a frame to be aggregated it has to have a depth >= minDepth. So, if minDepth
		// is set to 1, it means all the root frames will not be part of the aggregation.
		minDepth           int
		filterSystemFrames bool
	}

	FunctionMetric struct {
		index         int
		Fingerprint   uint32
		Function      string
		Package       string
		InApp         bool
		DurationsNS   []uint64
		SumDurationNS uint64
		SumSelfTimeNS uint64
		SampleCount   uint64
		Examples      []examples.ExampleMetadata
		WorstSelfTime uint64
		WorstExample  examples.ExampleMetadata
	}
)

func NewAggregator(
	maxUniqueFunctions int,
	maxNumberOfExamples int,
	minDepth int,
	filterSystemFrames bool,
) Aggregator {
	return Aggregator{
		callTreeFunctionsMap: make(map[uint32]*FunctionMetric),
		callTreeFunctions:    make([]*FunctionMetric, 0, maxUniqueFunctions+1),
		maxUniqueFunctions:   maxUniqueFunctions,
		maxNumberOfExamples:  maxNumberOfExamples,
		minDepth:             minDepth,
		filterSystemFrames:   filterSystemFrames,
	}
}

func (a Aggregator) Len() int {
	return len(a.callTreeFunctions)
}

func (a Aggregator) Less(i, j int) bool {
	fi := a.callTreeFunctions[i]
	fj := a.callTreeFunctions[j]

	if fi.SumSelfTimeNS != fj.SumSelfTimeNS {
		return fi.SumSelfTimeNS < fj.SumSelfTimeNS
	}

	if fi.SumDurationNS != fj.SumDurationNS {
		return fi.SumDurationNS < fj.SumDurationNS
	}

	return i < j
}

func (a *Aggregator) Swap(i, j int) {
	a.callTreeFunctions[i], a.callTreeFunctions[j] = a.callTreeFunctions[j], a.callTreeFunctions[i]

	// make sure to update the index so it can be easily found later
	a.callTreeFunctions[i].index = i
	a.callTreeFunctions[j].index = j
}

func (a *Aggregator) Push(item any) {
	n := a.Len()
	metric := item.(*FunctionMetric)
	metric.index = n
	a.callTreeFunctions = append(a.callTreeFunctions, metric)
	a.callTreeFunctionsMap[metric.Fingerprint] = metric
}

func (a *Aggregator) Pop() any {
	n := a.Len()
	item := a.callTreeFunctions[n-1]
	item.index = -1 // for safety
	a.callTreeFunctions = a.callTreeFunctions[0 : n-1]
	delete(a.callTreeFunctionsMap, item.Fingerprint)
	return item
}

func (a *Aggregator) AddFunction(n *nodetree.Node, depth int) {
	if a.filterSystemFrames && !n.IsApplication {
		return
	}

	if depth < a.minDepth {
		return
	}

	if !nodetree.ShouldAggregateFrame(n.Frame) {
		return
	}

	var function *FunctionMetric
	var ok bool
	fingerprint := n.Frame.Fingerprint()
	if function, ok = a.callTreeFunctionsMap[fingerprint]; ok {
		function.DurationsNS = append(function.DurationsNS, n.DurationsNS...)
		function.SumDurationNS += n.DurationNS
		function.SumSelfTimeNS += n.SelfTimeNS
		function.SampleCount += uint64(n.SampleCount)
		heap.Fix(a, function.index)
	} else {
		f := FunctionMetric{
			Fingerprint:   fingerprint,
			Function:      n.Name,
			Package:       n.Package,
			InApp:         n.IsApplication,
			DurationsNS:   n.DurationsNS,
			SumDurationNS: n.DurationNS,
			SumSelfTimeNS: n.SelfTimeNS,
			SampleCount:   uint64(n.SampleCount),
		}
		function = &f
		heap.Push(a, function)
	}

	for example := range n.Profiles {
		if len(function.Examples) < a.maxUniqueFunctions {
			function.Examples = append(function.Examples, example)
		}
	}

	if function.WorstSelfTime < n.WorstSelfTime {
		function.WorstSelfTime = n.WorstSelfTime
		function.WorstExample = n.WorstProfile
	}

	for a.Len() > a.maxUniqueFunctions {
		heap.Pop(a)
	}
}

func (a Aggregator) ToMetrics() []examples.FunctionMetrics {
	metrics := make([]examples.FunctionMetrics, 0, len(a.callTreeFunctions))

	for _, f := range a.callTreeFunctions {
		if f.SumSelfTimeNS <= 0 {
			continue
		}
		sort.Slice(f.Examples, func(i, j int) bool {
			example1 := f.Examples[i]
			example2 := f.Examples[j]

			if example1.ProfileID != "" {
				return example1.ProfileID < example2.ProfileID
			}

			if example2.ProfileID != "" {
				return true
			}

			return example1.ProfilerID < example2.ProfilerID
		})
		sort.Slice(f.DurationsNS, func(i, j int) bool {
			return f.DurationsNS[i] < f.DurationsNS[j]
		})

		p75, _ := quantile(f.DurationsNS, 0.75)
		p95, _ := quantile(f.DurationsNS, 0.95)
		p99, _ := quantile(f.DurationsNS, 0.99)

		m := examples.FunctionMetrics{
			Fingerprint: f.Fingerprint,
			Name:        f.Function,
			Package:     f.Package,
			InApp:       f.InApp,
			P75:         p75,
			P95:         p95,
			P99:         p99,
			Avg:         float64(f.SumDurationNS) / float64(len(f.DurationsNS)),
			Sum:         f.SumDurationNS,
			SumSelfTime: f.SumSelfTimeNS,
			Count:       f.SampleCount,
			Worst:       f.WorstExample,
			Examples:    f.Examples,
		}
		metrics = append(metrics, m)
	}

	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].Sum != metrics[j].Sum {
			return metrics[i].Sum > metrics[j].Sum
		}

		if metrics[i].SumSelfTime != metrics[j].SumSelfTime {
			return metrics[i].SumSelfTime > metrics[j].SumSelfTime
		}

		return i < j
	})

	return metrics
}

func quantile(values []uint64, q float64) (uint64, error) {
	if len(values) == 0 {
		return 0, errors.New("cannot compute percentile from empty list")
	}
	if q <= 0 || q > 1.0 {
		return 0, errors.New("q must be a value between 0 and 1.0")
	}
	index := int(math.Ceil(float64(len(values))*q)) - 1
	return values[index], nil
}

func ExtractFunctionsFromCallTreesForThread(
	callTreesForThread []*nodetree.Node,
	minDepth uint,
) []nodetree.CallTreeFunction {
	functions := make(map[uint32]nodetree.CallTreeFunction, 0)

	for _, callTree := range callTreesForThread {
		callTree.CollectFunctions(functions, "", 0, minDepth)
	}

	return mergeAndSortFunctions(functions)
}

func ExtractFunctionsFromCallTrees[T comparable](
	callTrees map[T][]*nodetree.Node,
	minDepth uint,
) []nodetree.CallTreeFunction {
	functions := make(map[uint32]nodetree.CallTreeFunction, 0)
	for tid, callTreesForThread := range callTrees {
		threadID := ""
		if t, ok := any(tid).(string); ok {
			threadID = t
		} else if t, ok := any(tid).(uint64); ok {
			threadID = strconv.FormatUint(t, 10)
		}
		for _, callTree := range callTreesForThread {
			callTree.CollectFunctions(functions, threadID, 0, minDepth)
		}
	}

	return mergeAndSortFunctions(functions)
}

func mergeAndSortFunctions(
	functions map[uint32]nodetree.CallTreeFunction,
) []nodetree.CallTreeFunction {
	functionsList := make([]nodetree.CallTreeFunction, 0, len(functions))
	for _, function := range functions {
		// We collect all functions in order to collect all durations.
		// If at the end, the self time is still 0, then it is omitted from the results.
		if function.SumSelfTimeNS == 0 {
			continue
		}
		if function.SampleCount <= 1 {
			// if there's only ever a single sample for this function in
			// the profile, we skip over it to reduce the amount of data
			continue
		}
		functionsList = append(functionsList, function)
	}

	// sort the list in descending order, and take the top N results
	sort.SliceStable(functionsList, func(i, j int) bool {
		if functionsList[i].SumSelfTimeNS != functionsList[j].SumSelfTimeNS {
			return functionsList[i].SumSelfTimeNS > functionsList[j].SumSelfTimeNS
		}
		return functionsList[i].SumDurationNS > functionsList[j].SumDurationNS
	})

	return functionsList
}

func CapAndFilterFunctions(functions []nodetree.CallTreeFunction, maxUniqueFunctionsPerProfile int, filterSystemFrames bool) []nodetree.CallTreeFunction {
	if !filterSystemFrames {
		if len(functions) > maxUniqueFunctionsPerProfile {
			return functions[:maxUniqueFunctionsPerProfile]
		}
		return functions
	}
	appFunctions := make([]nodetree.CallTreeFunction, 0, min(maxUniqueFunctionsPerProfile, len(functions)))
	for _, f := range functions {
		if !f.InApp {
			continue
		}
		appFunctions = append(appFunctions, f)
		if len(appFunctions) == maxUniqueFunctionsPerProfile {
			break
		}
	}
	return appFunctions
}
