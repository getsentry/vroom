package metrics

import (
	"context"
	"errors"
	"math"
	"sort"

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
		WorstID  string
		Examples []string
	}

	Aggregator struct {
		MaxUniqueFunctions uint
		MaxNumOfExamples   uint
		CallTreeFunctions  map[uint32]nodetree.CallTreeFunction
		FunctionsMetadata  map[uint32]FunctionsMetadata
	}

	FunctionMetrics struct {
		Name        string   `json:"name"`
		Package     string   `json:"package"`
		Fingerprint uint64   `json:"fingerprint"`
		InApp       bool     `json:"in_app"`
		P75         uint64   `json:"p75"`
		P95         uint64   `json:"p95"`
		P99         uint64   `json:"p99"`
		Avg         float64  `json:"avg"`
		Sum         uint64   `json:"sum"`
		Count       uint64   `json:"count"`
		Worst       string   `json:"worst"`
		Examples    []string `json:"examples"`
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

func (ma *Aggregator) AddFunctions(functions []nodetree.CallTreeFunction, ID string) {
	for _, f := range functions {
		if fn, ok := ma.CallTreeFunctions[f.Fingerprint]; ok {
			fn.SampleCount += f.SampleCount
			fn.SelfTimesNS = append(fn.SelfTimesNS, f.SelfTimesNS...)
			fn.SumSelfTimeNS += f.SumSelfTimeNS
			funcMetadata := ma.FunctionsMetadata[f.Fingerprint]
			if f.SumSelfTimeNS > funcMetadata.MaxVal {
				funcMetadata.MaxVal = f.SumSelfTimeNS
				funcMetadata.WorstID = ID
			}
			if len(funcMetadata.Examples) < int(ma.MaxNumOfExamples) {
				funcMetadata.Examples = append(funcMetadata.Examples, ID)
			}
			ma.FunctionsMetadata[f.Fingerprint] = funcMetadata
			ma.CallTreeFunctions[f.Fingerprint] = fn
		} else {
			ma.CallTreeFunctions[f.Fingerprint] = f
			ma.FunctionsMetadata[f.Fingerprint] = FunctionsMetadata{
				MaxVal:   f.SumSelfTimeNS,
				WorstID:  ID,
				Examples: []string{ID},
			}
		}
	}
}

func (ma *Aggregator) ToMetrics() []FunctionMetrics {
	metrics := make([]FunctionMetrics, 0, len(ma.CallTreeFunctions))

	for _, f := range ma.CallTreeFunctions {
		sort.Slice(f.SelfTimesNS, func(i, j int) bool {
			return f.SelfTimesNS[i] < f.SelfTimesNS[j]
		})
		p75, _ := quantile(f.SelfTimesNS, 0.75)
		p95, _ := quantile(f.SelfTimesNS, 0.95)
		p99, _ := quantile(f.SelfTimesNS, 0.99)
		metrics = append(metrics, FunctionMetrics{
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
			Worst:       ma.FunctionsMetadata[f.Fingerprint].WorstID,
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

func ExtractFunctionsFromCallTrees(
	callTrees map[uint64][]*nodetree.Node,
) []nodetree.CallTreeFunction {
	functions := make(map[uint32]nodetree.CallTreeFunction, 0)

	for _, callTreesForThread := range callTrees {
		for _, callTree := range callTreesForThread {
			callTree.CollectFunctions(functions)
		}
	}

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

func GetMetricsFromCandidates(
	ctx context.Context,
	storage *blob.Bucket,
	organizationID uint64,
	transactionProfileCandidates []utils.TransactionProfileCandidate,
	continuousProfileCandidates []utils.ContinuousProfileCandidate,
	jobs chan storageutil.ReadJob,
	maxUniqueFunctions uint,
	maxNumOfExamples uint,
) ([]FunctionMetrics, error) {
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
			ThreadID:       candidate.ThreadID,
			Start:          candidate.Start,
			End:            candidate.End,
			Storage:        storage,
			Result:         results,
		}
	}

	ma := NewAggregator(maxUniqueFunctions, maxNumOfExamples)

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

		if result, ok := res.(profile.ReadJobResult); ok {
			profileCallTrees, err := result.Profile.CallTrees()
			if err != nil {
				hub.CaptureException(err)
				continue
			}

			functions := CapAndFilterFunctions(ExtractFunctionsFromCallTrees(profileCallTrees), int(ma.MaxUniqueFunctions), true)
			ma.AddFunctions(functions, result.Profile.ID())
		} else if result, ok := res.(chunk.ReadJobResult); ok {
			chunkCallTrees, err := result.Chunk.CallTrees(result.ThreadID)
			if err != nil {
				hub.CaptureException(err)
				continue
			}
			intChunkCallTrees := make(map[uint64][]*nodetree.Node)
			var i uint64
			for _, v := range chunkCallTrees {
				// real TID here doesn't really matter as it's then
				// discarded (not used) by ExtractFunctionsFromCallTrees.
				// Here we're only assigning a random uint to make it compatible
				// with ExtractFunctionsFromCallTrees which expects an
				// uint64 -> []*nodetree.Node based on sample V1 int tid
				// the TID.
				//
				// We could even refactor ExtractFunctionsFromCallTrees
				// to simply accept []*nodetree.Node instead of a map
				// but we'd end up moving the iteration from map to a slice
				// somewhere else in the code.
				intChunkCallTrees[i] = v
				i++
			}
			functions := CapAndFilterFunctions(ExtractFunctionsFromCallTrees(intChunkCallTrees), int(ma.MaxUniqueFunctions), true)
			ma.AddFunctions(functions, result.Chunk.ID)
		} else {
			// this should never happen
			return nil, errors.New("unexpected result from storage")
		}
	}

	return ma.ToMetrics(), nil
}
