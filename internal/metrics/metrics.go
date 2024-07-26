package metrics

import (
	"errors"
	"math"
	"sort"

	"github.com/getsentry/vroom/internal/nodetree"
)

type FunctionsMetadata struct {
	MaxVal   uint64
	WorstID  string
	Examples []string
}

type Aggregator struct {
	MaxUniqueFunctions uint
	MaxNumOfExamples   uint
	CallTreeFunctions  map[uint32]nodetree.CallTreeFunction
	FunctionsMetadata  map[uint32]FunctionsMetadata
}

type FunctionMetrics struct {
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
	if q <= 0 || q > 100 {
		return 0, errors.New("q must be a value between 0 and 1.0")
	}
	index := int(math.Ceil(float64(len(values))*q)) - 1
	return values[index], nil
}
