package metrics

import (
	"sort"
	"testing"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/testutil"
	"github.com/getsentry/vroom/internal/utils"
)

func TestAggregatorAddFunctions(t *testing.T) {
	tests := []struct {
		name              string
		calltreeFunctions []nodetree.CallTreeFunction
		want              Aggregator
	}{
		{
			name: "addFunctions",
			calltreeFunctions: []nodetree.CallTreeFunction{
				{
					Function:      "a",
					Fingerprint:   0,
					SelfTimesNS:   []uint64{10, 5, 25},
					SumSelfTimeNS: 40,
				},
				{
					Function:      "b",
					Fingerprint:   1,
					SelfTimesNS:   []uint64{45, 60},
					SumSelfTimeNS: 105,
				},
			},
			want: Aggregator{
				MaxUniqueFunctions: 100,
				MaxNumOfExamples:   5,
				CallTreeFunctions: map[uint32]nodetree.CallTreeFunction{
					0: {
						Function:      "a",
						Fingerprint:   0,
						SelfTimesNS:   []uint64{10, 5, 25, 10, 5, 25},
						SumSelfTimeNS: 80,
					},
					1: {
						Function:      "b",
						Fingerprint:   1,
						SelfTimesNS:   []uint64{45, 60, 45, 60},
						SumSelfTimeNS: 210,
					},
				},
				FunctionsMetadata: map[uint32]FunctionsMetadata{
					0: {
						MaxVal:   40,
						Worst:    utils.ExampleMetadata{ProfileID: "1"},
						Examples: []utils.ExampleMetadata{{ProfileID: "1"}, {ProfileID: "2"}},
					},
					1: {
						MaxVal:   105,
						Worst:    utils.ExampleMetadata{ProfileID: "1"},
						Examples: []utils.ExampleMetadata{{ProfileID: "1"}, {ProfileID: "2"}},
					},
				}, // end want
			},
		}, // end first test
	} // end tests list

	ma := NewAggregator(100, 5, 0)
	for _, test := range tests {
		// add the same calltreeFunctions twice: once coming from a profile/chunk with
		// ID 1 and the second one with ID 2
		ma.AddFunctions(test.calltreeFunctions, utils.ExampleMetadata{ProfileID: "1"})
		ma.AddFunctions(test.calltreeFunctions, utils.ExampleMetadata{ProfileID: "2"})
		if diff := testutil.Diff(ma, test.want); diff != "" {
			t.Fatalf("Result mismatch: got - want +\n%s", diff)
		}
	}
}

func TestAggregatorToMetrics(t *testing.T) {
	tests := []struct {
		name       string
		Aggregator Aggregator
		want       []utils.FunctionMetrics
	}{
		{
			name: "toMetrics",
			Aggregator: Aggregator{
				MaxUniqueFunctions: 100,
				CallTreeFunctions: map[uint32]nodetree.CallTreeFunction{
					0: {
						Function:      "a",
						Fingerprint:   0,
						SelfTimesNS:   []uint64{1, 2, 3, 4, 10, 8, 7, 11, 20},
						SumSelfTimeNS: 66,
						SampleCount:   2,
					},
					1: {
						Function:      "b",
						Fingerprint:   1,
						SelfTimesNS:   []uint64{1, 2, 3, 4, 10, 8, 7, 11, 20},
						SumSelfTimeNS: 66,
						SampleCount:   2,
					},
				}, //end callTreeFunctions
				FunctionsMetadata: map[uint32]FunctionsMetadata{
					0: {
						MaxVal:   66,
						Worst:    utils.ExampleMetadata{ProfileID: "1"},
						Examples: []utils.ExampleMetadata{{ProfileID: "1"}, {ProfileID: "2"}},
					},
					1: {
						MaxVal:   66,
						Worst:    utils.ExampleMetadata{ProfileID: "3"},
						Examples: []utils.ExampleMetadata{{ProfileID: "1"}, {ProfileID: "3"}},
					},
				}, //end functionsMetadata
			}, //end Aggregator
			want: []utils.FunctionMetrics{
				{
					Name:        "a",
					Fingerprint: 0,
					P75:         10,
					P95:         20,
					P99:         20,
					Count:       2,
					Sum:         66,
					Avg:         float64(66) / float64(9),
					Worst:       utils.ExampleMetadata{ProfileID: "1"},
					Examples:    []utils.ExampleMetadata{{ProfileID: "1"}, {ProfileID: "2"}},
				},
				{
					Name:        "b",
					Fingerprint: 1,
					P75:         10,
					P95:         20,
					P99:         20,
					Count:       2,
					Sum:         66,
					Avg:         float64(66) / float64(9),
					Worst:       utils.ExampleMetadata{ProfileID: "3"},
					Examples:    []utils.ExampleMetadata{{ProfileID: "1"}, {ProfileID: "3"}},
				},
			}, //want
		},
	}

	for _, test := range tests {
		metrics := test.Aggregator.ToMetrics()
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].Name < metrics[j].Name
		})
		if diff := testutil.Diff(metrics, test.want); diff != "" {
			t.Fatalf("Result mismatch: got - want +\n%s", diff)
		}
	}
}
