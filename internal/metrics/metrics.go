package metrics

import (
	"context"
	"errors"
	"math"
	"sort"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/chunk"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/storageutil"
	"github.com/getsentry/vroom/internal/utils"
	"gocloud.dev/blob"
)

type (
	FunctionsMetadata struct {
		MaxVal   uint64
		Worst    utils.ExampleMetadata
		Examples []utils.ExampleMetadata
	}

	Aggregator struct {
		MaxUniqueFunctions uint
		MaxNumOfExamples   uint
		CallTreeFunctions  map[uint32]nodetree.CallTreeFunction
		FunctionsMetadata  map[uint32]FunctionsMetadata
	}
)

func NewAggregator(MaxUniqueFunctions uint, MaxNumOfExamples uint) Aggregator {
	return Aggregator{
		MaxUniqueFunctions: MaxUniqueFunctions,
		MaxNumOfExamples:   MaxNumOfExamples,
		CallTreeFunctions:  make(map[uint32]nodetree.CallTreeFunction),
		FunctionsMetadata:  make(map[uint32]FunctionsMetadata),
	}
}

func (ma *Aggregator) AddFunctions(functions []nodetree.CallTreeFunction, resultMetadata utils.ExampleMetadata) {
	for _, f := range functions {
		if fn, ok := ma.CallTreeFunctions[f.Fingerprint]; ok {
			fn.SampleCount += f.SampleCount
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
				Examples: []utils.ExampleMetadata{resultMetadata},
			}
		}
	}
}

func (ma *Aggregator) ToMetrics() []utils.FunctionMetrics {
	metrics := make([]utils.FunctionMetrics, 0, len(ma.CallTreeFunctions))

	for _, f := range ma.CallTreeFunctions {
		sort.Slice(f.SelfTimesNS, func(i, j int) bool {
			return f.SelfTimesNS[i] < f.SelfTimesNS[j]
		})
		p75, _ := quantile(f.SelfTimesNS, 0.75)
		p95, _ := quantile(f.SelfTimesNS, 0.95)
		p99, _ := quantile(f.SelfTimesNS, 0.99)
		metrics = append(metrics, utils.FunctionMetrics{
			Name:        f.Function,
			Package:     f.Package,
			Fingerprint: uint64(f.Fingerprint),
			InApp:       f.InApp,
			P75:         p75,
			P95:         p95,
			P99:         p99,
			Avg:         float64(f.SumSelfTimeNS) / float64(len(f.SelfTimesNS)),
			Sum:         f.SumSelfTimeNS,
			Count:       uint64(f.SampleCount),
			Worst:       ma.FunctionsMetadata[f.Fingerprint].Worst,
			Examples:    ma.FunctionsMetadata[f.Fingerprint].Examples,
		})
	}
	sort.Slice(metrics, func(i, j int) bool {
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
) []nodetree.CallTreeFunction {
	functions := make(map[uint32]nodetree.CallTreeFunction, 0)

	for _, callTree := range callTreesForThread {
		callTree.CollectFunctions(functions, "")
	}

	return mergeAndSortFunctions(functions)
}

func ExtractFunctionsFromCallTrees[T comparable](
	callTrees map[T][]*nodetree.Node,
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
			callTree.CollectFunctions(functions, threadID)
		}
	}

	return mergeAndSortFunctions(functions)
}

func mergeAndSortFunctions(
	functions map[uint32]nodetree.CallTreeFunction,
) []nodetree.CallTreeFunction {
	functionsList := make([]nodetree.CallTreeFunction, 0, len(functions))
	for _, function := range functions {
		if function.SampleCount <= 1 {
			// if there's only ever a single sample for this function in
			// the profile, we skip over it to reduce the amount of data
			continue
		}
		functionsList = append(functionsList, function)
	}

	// sort the list in descending order, and take the top N results
	sort.SliceStable(functionsList, func(i, j int) bool {
		return functionsList[i].SumSelfTimeNS > functionsList[j].SumSelfTimeNS
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
	transactionProfileCandidates []utils.TransactionProfileCandidate,
	continuousProfileCandidates []utils.ContinuousProfileCandidate,
	jobs chan storageutil.ReadJob,
) ([]utils.FunctionMetrics, error) {
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
			Storage:        storage,
			Result:         results,
		}
	}

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

		var resultMetadata utils.ExampleMetadata
		if result, ok := res.(profile.ReadJobResult); ok {
			profileCallTrees, err := result.Profile.CallTrees()
			if err != nil {
				hub.CaptureException(err)
				continue
			}
			resultMetadata = utils.NewExampleFromProfileID(result.Profile.ProjectID(), result.Profile.ID())
			functions := CapAndFilterFunctions(ExtractFunctionsFromCallTrees(profileCallTrees), int(ma.MaxUniqueFunctions), true)
			ma.AddFunctions(functions, resultMetadata)
		} else if result, ok := res.(chunk.ReadJobResult); ok {
			chunkCallTrees, err := result.Chunk.CallTrees(result.ThreadID)
			if err != nil {
				hub.CaptureException(err)
				continue
			}

			resultMetadata = utils.NewExampleFromProfilerChunk(
				result.Chunk.GetProjectID(),
				result.Chunk.GetProfilerID(),
				result.Chunk.GetID(),
				result.TransactionID,
				result.ThreadID,
				result.Start,
				result.End,
			)
			functions := CapAndFilterFunctions(ExtractFunctionsFromCallTrees(chunkCallTrees), int(ma.MaxUniqueFunctions), true)
			ma.AddFunctions(functions, resultMetadata)
		} else {
			// this should never happen
			return nil, errors.New("unexpected result from storage")
		}
	}

	return ma.ToMetrics(), nil
}
