package metrics

import (
	"context"
	"errors"
	"math"
	"sort"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/chunk"
	"github.com/getsentry/vroom/internal/examples"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/storageutil"
	"gocloud.dev/blob"
)

type (
	FunctionsMetadata struct {
		MaxVal   uint64
		Worst    examples.ExampleMetadata
		Examples []examples.ExampleMetadata
	}

	Aggregator struct {
		MaxUniqueFunctions uint
		// For a frame to be aggregated it has to have a depth >= MinDepth.
		// So, if MinDepth is set to 1, it means all the root frames will not be part of the aggregation.
		MinDepth          uint
		MaxNumOfExamples  uint
		CallTreeFunctions map[uint32]nodetree.CallTreeFunction
		FunctionsMetadata map[uint32]FunctionsMetadata
	}
)

func NewAggregator(MaxUniqueFunctions uint, MaxNumOfExamples uint, MinDepth uint) Aggregator {
	return Aggregator{
		MaxUniqueFunctions: MaxUniqueFunctions,
		MinDepth:           MinDepth,
		MaxNumOfExamples:   MaxNumOfExamples,
		CallTreeFunctions:  make(map[uint32]nodetree.CallTreeFunction),
		FunctionsMetadata:  make(map[uint32]FunctionsMetadata),
	}
}

func (ma *Aggregator) AddFunctions(functions []nodetree.CallTreeFunction, resultMetadata examples.ExampleMetadata) {
	for _, f := range functions {
		if fn, ok := ma.CallTreeFunctions[f.Fingerprint]; ok {
			fn.SampleCount += f.SampleCount
			fn.DurationsNS = append(fn.DurationsNS, f.DurationsNS...)
			fn.SumDurationNS += f.SumDurationNS
			fn.SelfTimesNS = append(fn.SelfTimesNS, f.SelfTimesNS...)
			fn.SumSelfTimeNS += f.SumSelfTimeNS
			funcMetadata := ma.FunctionsMetadata[f.Fingerprint]
			if f.SumSelfTimeNS > funcMetadata.MaxVal {
				funcMetadata.MaxVal = f.SumSelfTimeNS
				funcMetadata.Worst = resultMetadata
			}
			if len(funcMetadata.Examples) < int(ma.MaxNumOfExamples) {
				funcMetadata.Examples = append(funcMetadata.Examples, resultMetadata)
			}
			ma.FunctionsMetadata[f.Fingerprint] = funcMetadata
			ma.CallTreeFunctions[f.Fingerprint] = fn
		} else {
			ma.CallTreeFunctions[f.Fingerprint] = f
			ma.FunctionsMetadata[f.Fingerprint] = FunctionsMetadata{
				MaxVal:   f.SumSelfTimeNS,
				Worst:    resultMetadata,
				Examples: []examples.ExampleMetadata{resultMetadata},
			}
		}
	}
}

func (ma *Aggregator) ToMetrics() []examples.FunctionMetrics {
	metrics := make([]examples.FunctionMetrics, 0, len(ma.CallTreeFunctions))

	for _, f := range ma.CallTreeFunctions {
		sort.Slice(f.DurationsNS, func(i, j int) bool {
			return f.DurationsNS[i] < f.DurationsNS[j]
		})
		p75, _ := quantile(f.DurationsNS, 0.75)
		p95, _ := quantile(f.DurationsNS, 0.95)
		p99, _ := quantile(f.DurationsNS, 0.99)
		metrics = append(metrics, examples.FunctionMetrics{
			Name:        f.Function,
			Package:     f.Package,
			Fingerprint: f.Fingerprint,
			InApp:       f.InApp,
			P75:         p75,
			P95:         p95,
			P99:         p99,
			Avg:         float64(f.SumDurationNS) / float64(len(f.DurationsNS)),
			Sum:         f.SumDurationNS,
			SumSelfTime: f.SumSelfTimeNS,
			Count:       uint64(f.SampleCount),
			Worst:       ma.FunctionsMetadata[f.Fingerprint].Worst,
			Examples:    ma.FunctionsMetadata[f.Fingerprint].Examples,
		})
	}
	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].SumSelfTime != metrics[j].SumSelfTime {
			return metrics[i].SumSelfTime > metrics[j].SumSelfTime
		}
		return metrics[i].Sum > metrics[j].Sum
	})
	if len(metrics) > int(ma.MaxUniqueFunctions) {
		metrics = metrics[:ma.MaxUniqueFunctions]
	}
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

func (ma *Aggregator) GetMetricsFromCandidates(
	ctx context.Context,
	storage *blob.Bucket,
	organizationID uint64,
	transactionProfileCandidates []examples.TransactionProfileCandidate,
	continuousProfileCandidates []examples.ContinuousProfileCandidate,
	jobs chan storageutil.ReadJob,
) ([]examples.FunctionMetrics, error) {
	hub := sentry.GetHubFromContext(ctx)

	results := make(chan storageutil.ReadJobResult)
	defer close(results)

	go func() {
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
				Storage:        storage,
				Result:         results,
			}
		}
	}()

	numCandidates := len(transactionProfileCandidates) + len(continuousProfileCandidates)

	for i := 0; i < numCandidates; i++ {
		res := <-results

		err := res.Error()
		if err != nil {
			if errors.Is(err, storageutil.ErrObjectNotFound) {
				continue
			}
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			if hub != nil {
				hub.CaptureException(err)
			}
			continue
		}

		var resultMetadata examples.ExampleMetadata
		if result, ok := res.(profile.ReadJobResult); ok {
			profileCallTrees, err := result.Profile.CallTrees()
			if err != nil {
				hub.CaptureException(err)
				continue
			}
			start, end := result.Profile.StartAndEndEpoch()
			resultMetadata = examples.NewExampleFromProfileID(
				result.Profile.ProjectID(),
				result.Profile.ID(),
				start,
				end,
			)
			functions := CapAndFilterFunctions(ExtractFunctionsFromCallTrees(profileCallTrees, ma.MinDepth), int(ma.MaxUniqueFunctions), true)
			ma.AddFunctions(functions, resultMetadata)
		} else if result, ok := res.(chunk.ReadJobResult); ok {
			chunkCallTrees, err := result.Chunk.CallTrees(result.ThreadID)
			if err != nil {
				hub.CaptureException(err)
				continue
			}

			resultMetadata = examples.NewExampleFromProfilerChunk(
				result.Chunk.GetProjectID(),
				result.Chunk.GetProfilerID(),
				result.Chunk.GetID(),
				result.TransactionID,
				result.ThreadID,
				result.Start,
				result.End,
			)
			functions := CapAndFilterFunctions(ExtractFunctionsFromCallTrees(chunkCallTrees, ma.MinDepth), int(ma.MaxUniqueFunctions), true)
			ma.AddFunctions(functions, resultMetadata)
		} else {
			// this should never happen
			return nil, errors.New("unexpected result from storage")
		}
	}

	return ma.ToMetrics(), nil
}
