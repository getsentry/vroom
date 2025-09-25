package flamegraph

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/getsentry/vroom/internal/examples"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/metrics"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/testutil"
)

func TestFlamegraphAggregation(t *testing.T) {
	tests := []struct {
		name     string
		profiles []sample.Profile
		output   speedscope.Output
		metrics  []examples.FunctionMetrics
	}{
		{
			name: "Basic profiles aggregation",
			profiles: []sample.Profile{
				{
					RawProfile: sample.RawProfile{
						EventID:  "ab1",
						Platform: platform.Cocoa,
						Version:  "1",
						Trace: sample.Trace{
							Frames: []frame.Frame{
								{
									Function: "a",
									Package:  "test.package",
									InApp:    &testutil.False,
								},
								{
									Function: "b",
									Package:  "test.package",
									InApp:    &testutil.False,
								},
								{
									Function: "c",
									Package:  "test.package",
									InApp:    &testutil.True,
								},
							}, // end frames
							Stacks: []sample.Stack{
								{1, 0}, // b,a
								{2},    // c
								{0},    // a
							},
							Samples: []sample.Sample{
								{
									ElapsedSinceStartNS: 0,
									StackID:             0,
								},
								{
									ElapsedSinceStartNS: 10,
									StackID:             1,
								},
								{
									ElapsedSinceStartNS: 20,
									StackID:             0,
								},
								{
									ElapsedSinceStartNS: 30,
									StackID:             0,
								},
								{
									ElapsedSinceStartNS: 40,
									StackID:             2,
								},
							}, // end Samples
						}, // end Trace
					},
				}, // end prof definition
				{
					RawProfile: sample.RawProfile{
						EventID:  "cd2",
						Platform: platform.Cocoa,
						Version:  "1",
						Trace: sample.Trace{
							Frames: []frame.Frame{
								{
									Function: "a",
									Package:  "test.package",
									InApp:    &testutil.False,
								},
								{
									Function: "c",
									Package:  "test.package",
									InApp:    &testutil.True,
								},
								{
									Function: "e",
									Package:  "test.package",
									InApp:    &testutil.False,
								},
								{
									Function: "b",
									Package:  "test.package",
									InApp:    &testutil.False,
								},
							}, // end frames
							Stacks: []sample.Stack{
								{0, 1}, // a,c
								{2},    // e
								{3, 0}, // b,a
							},
							Samples: []sample.Sample{
								{
									ElapsedSinceStartNS: 0,
									StackID:             0,
								},
								{
									ElapsedSinceStartNS: 10,
									StackID:             1,
								},
								{
									ElapsedSinceStartNS: 20,
									StackID:             2,
								},
								{
									ElapsedSinceStartNS: 30,
									StackID:             2,
								},
							}, // end Samples
						}, // end Trace
					},
				}, // end prof definition
			},
			output: speedscope.Output{
				Metadata: speedscope.ProfileMetadata{
					ProfileView: speedscope.ProfileView{
						ProjectID: 99,
					},
				},
				Profiles: []interface{}{
					speedscope.SampledProfile{
						EndValue:     7,
						IsMainThread: true,
						Samples: [][]int{
							{2},
							{3},
							{2, 0},
							{0, 1},
						},
						SamplesExamples: [][]int{
							{0},
							{1},
							{1},
							{0, 1},
						},
						Type:              "sampled",
						Unit:              "count",
						Weights:           []uint64{1, 1, 1, 4},
						SampleCounts:      []uint64{1, 1, 1, 4},
						SampleDurationsNs: []uint64{10, 10, 10, 40},
					},
				},
				Shared: speedscope.SharedData{
					Frames: []speedscope.Frame{
						{Image: "test.package", Name: "a", Fingerprint: 2430275452},
						{Image: "test.package", Name: "b", Fingerprint: 2430275455},
						{Image: "test.package", Name: "c", Fingerprint: 2430275454, IsApplication: true},
						{Image: "test.package", Name: "e", Fingerprint: 2430275448},
					},
					FrameInfos: []speedscope.FrameInfo{
						{Count: 4, Weight: 50},
						{Count: 3, Weight: 40},
						{Count: 2, Weight: 20},
						{Count: 1, Weight: 10},
					},
					Profiles: []examples.ExampleMetadata{
						{ProfileID: "ab1"},
						{ProfileID: "cd2"},
					},
				},
			},
			metrics: []examples.FunctionMetrics{
				{
					Name:        "a",
					Package:     "test.package",
					Fingerprint: 2430275452,
					P75:         10,
					P95:         20,
					P99:         20,
					Avg:         float64(50) / float64(4),
					Sum:         50,
					SumSelfTime: 10,
					Count:       5,
					Worst:       examples.ExampleMetadata{ProfileID: "cd2"},
					Examples:    []examples.ExampleMetadata{{ProfileID: "cd2"}},
				},
				{
					Name:        "b",
					Package:     "test.package",
					Fingerprint: 2430275455,
					P75:         20,
					P95:         20,
					P99:         20,
					Avg:         float64(40) / float64(3),
					Sum:         40,
					SumSelfTime: 40,
					Count:       4,
					Worst:       examples.ExampleMetadata{ProfileID: "ab1"},
					Examples:    []examples.ExampleMetadata{{ProfileID: "ab1"}, {ProfileID: "cd2"}},
				},
				{
					Name:        "c",
					Package:     "test.package",
					Fingerprint: 2430275454,
					InApp:       true,
					P75:         10,
					P95:         10,
					P99:         10,
					Avg:         10,
					Sum:         20,
					SumSelfTime: 20,
					Count:       2,
					Worst:       examples.ExampleMetadata{ProfileID: "ab1"},
					Examples:    []examples.ExampleMetadata{{ProfileID: "ab1"}},
				},
				{
					Name:        "e",
					Package:     "test.package",
					Fingerprint: 2430275448,
					P75:         10,
					P95:         10,
					P99:         10,
					Avg:         10,
					Sum:         10,
					SumSelfTime: 10,
					Count:       1,
					Worst:       examples.ExampleMetadata{ProfileID: "cd2"},
					Examples:    []examples.ExampleMetadata{{ProfileID: "cd2"}},
				},
			},
		},
		{
			name: "Complex profiles aggregation",
			profiles: []sample.Profile{
				{
					RawProfile: sample.RawProfile{
						EventID:  "ab1",
						Platform: platform.Cocoa,
						Version:  "1",
						Trace: sample.Trace{
							Frames: []frame.Frame{
								{
									Function: "a",
									Package:  "test.package",
									InApp:    &testutil.True,
								},
								{
									Function: "b",
									Package:  "test.package",
									InApp:    &testutil.True,
								},
								{
									Function: "c",
									Package:  "test.package",
									InApp:    &testutil.True,
								},
							}, // end frames
							Stacks: []sample.Stack{
								{2, 1, 0}, // c,b,a
								{1, 0},    // b,a
								{0},       // a
							},
							Samples: []sample.Sample{
								{
									ElapsedSinceStartNS: 0,
									StackID:             0,
								},
								{
									ElapsedSinceStartNS: 10,
									StackID:             0,
								},
								{
									ElapsedSinceStartNS: 20,
									StackID:             1,
								},
								{
									ElapsedSinceStartNS: 30,
									StackID:             1,
								},
								{
									ElapsedSinceStartNS: 40,
									StackID:             2,
								},
								{
									ElapsedSinceStartNS: 50,
									StackID:             2,
								},
								{
									ElapsedSinceStartNS: 60,
									StackID:             0,
								},
								{
									ElapsedSinceStartNS: 70,
									StackID:             0,
								},
								{
									ElapsedSinceStartNS: 80,
									StackID:             0,
								},
								{
									ElapsedSinceStartNS: 90,
									StackID:             0,
								},
							}, // end Samples
						}, // end Trace
					},
				}, // end prof definition
			},
			output: speedscope.Output{
				Metadata: speedscope.ProfileMetadata{
					ProfileView: speedscope.ProfileView{
						ProjectID: 99,
					},
				},
				Profiles: []interface{}{
					speedscope.SampledProfile{
						EndValue:     9,
						IsMainThread: true,
						Samples: [][]int{
							{0},
							{0, 1, 2},
							{0, 1},
						},
						SamplesExamples: [][]int{
							{0},
							{0},
							{0},
						},
						Type:              "sampled",
						Unit:              "count",
						Weights:           []uint64{2, 5, 2},
						SampleCounts:      []uint64{2, 5, 2},
						SampleDurationsNs: []uint64{20, 50, 20},
					},
				},
				Shared: speedscope.SharedData{
					Frames: []speedscope.Frame{
						{Image: "test.package", Name: "a", Fingerprint: 2430275452, IsApplication: true},
						{Image: "test.package", Name: "b", Fingerprint: 2430275455, IsApplication: true},
						{Image: "test.package", Name: "c", Fingerprint: 2430275454, IsApplication: true},
					},
					FrameInfos: []speedscope.FrameInfo{
						{Count: 1, Weight: 90},
						{Count: 2, Weight: 70},
						{Count: 2, Weight: 50},
					},
					Profiles: []examples.ExampleMetadata{
						{ProfileID: "ab1"},
					},
				},
			},
			metrics: []examples.FunctionMetrics{
				{
					Name:        "a",
					Package:     "test.package",
					Fingerprint: 2430275452,
					InApp:       true,
					P75:         90,
					P95:         90,
					P99:         90,
					Avg:         90,
					Sum:         90,
					SumSelfTime: 20,
					Count:       9,
					Worst:       examples.ExampleMetadata{ProfileID: "ab1"},
					Examples:    []examples.ExampleMetadata{{ProfileID: "ab1"}},
				},
				{
					Name:        "b",
					Package:     "test.package",
					Fingerprint: 2430275455,
					InApp:       true,
					P75:         40,
					P95:         40,
					P99:         40,
					Avg:         35,
					Sum:         70,
					SumSelfTime: 20,
					Count:       7,
					Worst:       examples.ExampleMetadata{ProfileID: "ab1"},
					Examples:    []examples.ExampleMetadata{{ProfileID: "ab1"}},
				},
				{
					Name:        "c",
					Package:     "test.package",
					Fingerprint: 2430275454,
					InApp:       true,
					P75:         30,
					P95:         30,
					P99:         30,
					Avg:         25,
					Sum:         50,
					SumSelfTime: 50,
					Count:       5,
					Worst:       examples.ExampleMetadata{ProfileID: "ab1"},
					Examples:    []examples.ExampleMetadata{{ProfileID: "ab1"}},
				},
			},
		},
		{
			name: "zero self time",
			profiles: []sample.Profile{
				{
					RawProfile: sample.RawProfile{
						EventID:  "ab1",
						Platform: platform.Cocoa,
						Version:  "1",
						Trace: sample.Trace{
							Frames: []frame.Frame{
								{
									Function: "a",
									Package:  "test.package",
									InApp:    &testutil.True,
								},
								{
									Function: "b",
									Package:  "test.package",
									InApp:    &testutil.True,
								},
								{
									Function: "c",
									Package:  "test.package",
									InApp:    &testutil.True,
								},
							}, // end frames
							Stacks: []sample.Stack{
								{2, 1, 0}, // c,b,a
								{1},       // b
							},
							Samples: []sample.Sample{
								{
									ElapsedSinceStartNS: 0,
									StackID:             0,
								},
								{
									ElapsedSinceStartNS: 10,
									StackID:             0,
								},
								{
									ElapsedSinceStartNS: 20,
									StackID:             0,
								},
								{
									ElapsedSinceStartNS: 30,
									StackID:             1,
								},
								{
									ElapsedSinceStartNS: 40,
									StackID:             1,
								},
								{
									ElapsedSinceStartNS: 50,
									StackID:             1,
								},
							}, // end Samples
						}, // end Trace
					},
				}, // end prof definition
			},
			output: speedscope.Output{
				Metadata: speedscope.ProfileMetadata{
					ProfileView: speedscope.ProfileView{
						ProjectID: 99,
					},
				},
				Profiles: []interface{}{
					speedscope.SampledProfile{
						EndValue:     5,
						IsMainThread: true,
						Samples: [][]int{
							{1},
							{0, 1, 2},
						},
						SamplesExamples: [][]int{
							{0},
							{0},
						},
						Type:              "sampled",
						Unit:              "count",
						Weights:           []uint64{2, 3},
						SampleCounts:      []uint64{2, 3},
						SampleDurationsNs: []uint64{20, 30},
					},
				},
				Shared: speedscope.SharedData{
					Frames: []speedscope.Frame{
						{Image: "test.package", Name: "a", Fingerprint: 2430275452, IsApplication: true},
						{Image: "test.package", Name: "b", Fingerprint: 2430275455, IsApplication: true},
						{Image: "test.package", Name: "c", Fingerprint: 2430275454, IsApplication: true},
					},
					FrameInfos: []speedscope.FrameInfo{
						{Count: 1, Weight: 30},
						{Count: 2, Weight: 50},
						{Count: 1, Weight: 30},
					},
					Profiles: []examples.ExampleMetadata{
						{ProfileID: "ab1"},
					},
				},
			},
			metrics: []examples.FunctionMetrics{
				{
					Name:        "b",
					Package:     "test.package",
					Fingerprint: 2430275455,
					InApp:       true,
					P75:         30,
					P95:         30,
					P99:         30,
					Avg:         25,
					Sum:         50,
					SumSelfTime: 20,
					Count:       5,
					Worst:       examples.ExampleMetadata{ProfileID: "ab1"},
					Examples:    []examples.ExampleMetadata{{ProfileID: "ab1"}},
				},
				{
					Name:        "c",
					Package:     "test.package",
					Fingerprint: 2430275454,
					InApp:       true,
					P75:         30,
					P95:         30,
					P99:         30,
					Avg:         30,
					Sum:         30,
					SumSelfTime: 30,
					Count:       3,
					Worst:       examples.ExampleMetadata{ProfileID: "ab1"},
					Examples:    []examples.ExampleMetadata{{ProfileID: "ab1"}},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ft []*nodetree.Node

			for _, sp := range test.profiles {
				p := profile.New(&sp)
				callTrees, err := p.CallTrees()
				if err != nil {
					t.Fatalf("error when generating calltrees: %v", err)
				}
				example := examples.ExampleMetadata{
					ProfileID: p.ID(),
				}
				addCallTreeToFlamegraph(&ft, callTrees[0], annotateWithProfileExample(example))
			}

			if diff := testutil.Diff(toSpeedscope(context.TODO(), ft, 10, 99), test.output); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}

			ma := metrics.NewAggregator(100, 5, 0, false)
			for _, tree := range ft {
				tree.Visit(ma.AddFunction)
			}
			m := ma.ToMetrics()

			if diff := testutil.Diff(m, test.metrics); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestAnnotatingWithExamples(t *testing.T) {
	threadID := "0"

	tests := []struct {
		name      string
		callTrees []*nodetree.Node
		examples  []examples.ExampleMetadata
		output    speedscope.Output
	}{
		{
			name: "Annotate with profile id",
			callTrees: []*nodetree.Node{
				{
					DurationNS:    40_000_000,
					EndNS:         50_000_000,
					StartNS:       10_000_000,
					Fingerprint:   14164357600995800812,
					IsApplication: true,
					Name:          "function1",
					SampleCount:   2,
					Occurrence:    2,
					Frame:         frame.Frame{Function: "function1"},
					Profiles:      make(map[examples.ExampleMetadata]struct{}),
					Children: []*nodetree.Node{
						{
							DurationNS:    10_000_000,
							EndNS:         50_000_000,
							Fingerprint:   9531802423075301657,
							IsApplication: true,
							Name:          "function2",
							SampleCount:   1,
							Occurrence:    1,
							StartNS:       40_000_000,
							Frame:         frame.Frame{Function: "function2"},
							Profiles:      make(map[examples.ExampleMetadata]struct{}),
						},
					},
				},
			},
			examples: []examples.ExampleMetadata{
				examples.NewExampleFromProfileID(1, "2", 10_000_000, 50_000_000),
				examples.NewExampleFromProfilerChunk(3, "4", "5", "6", &threadID, 10_000_000, 50_000_000),
			},
			output: speedscope.Output{
				Metadata: speedscope.ProfileMetadata{
					ProfileView: speedscope.ProfileView{
						ProjectID: 99,
					},
				},
				Profiles: []interface{}{
					speedscope.SampledProfile{
						EndValue:     4,
						IsMainThread: true,
						Samples: [][]int{
							{0, 1},
							{0},
						},
						SamplesExamples:   [][]int{{0, 1}, {0, 1}},
						Type:              "sampled",
						Unit:              "count",
						Weights:           []uint64{2, 2},
						SampleCounts:      []uint64{2, 2},
						SampleDurationsNs: []uint64{20_000_000, 60_000_000},
					},
				},
				Shared: speedscope.SharedData{
					Frames: []speedscope.Frame{
						{Name: "function1", Fingerprint: 3932509230, IsApplication: true},
						{Name: "function2", Fingerprint: 3932509229, IsApplication: true},
					},
					FrameInfos: []speedscope.FrameInfo{
						{Count: 4, Weight: 80_000_000},
						{Count: 2, Weight: 20_000_000},
					},
					Profiles: []examples.ExampleMetadata{
						{
							ProjectID: 1,
							ProfileID: "2",
							Start:     0.01,
							End:       0.05,
						},
						{
							ProjectID:     3,
							ProfilerID:    "4",
							ChunkID:       "5",
							TransactionID: "6",
							ThreadID:      &threadID,
							Start:         0.01,
							End:           0.05,
						},
					},
				},
			},
		},
	}

	options := cmp.Options{
		// This option will order profile examples since we only want to compare values and not order.
		cmpopts.SortSlices(func(a, b string) bool {
			return a < b
		}),
		cmpopts.SortSlices(func(a, b int) bool {
			return a < b
		}),
		// This option will order profile IDs since we only want to compare values and not order.
		cmpopts.SortSlices(func(a, b examples.ExampleMetadata) bool {
			if a.ProjectID != b.ProjectID {
				return a.ProjectID < b.ProjectID
			}
			if a.ProfilerID != b.ProfilerID {
				return a.ProfilerID < b.ProfilerID
			}
			if a.ChunkID != b.ChunkID {
				return a.ChunkID < b.ChunkID
			}
			if a.TransactionID != b.TransactionID {
				return a.TransactionID < b.TransactionID
			}
			if a.Start != b.Start {
				return a.Start < b.Start
			}
			if a.End != b.End {
				return a.End < b.End
			}
			return a.ProfileID < b.ProfileID
		}),
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ft []*nodetree.Node
			for _, example := range test.examples {
				addCallTreeToFlamegraph(&ft, test.callTrees, annotateWithProfileExample(example))
			}
			if diff := testutil.Diff(toSpeedscope(context.TODO(), ft, 10, 99), test.output, options); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
