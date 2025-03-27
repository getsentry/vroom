package flamegraph

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/testutil"
	"github.com/getsentry/vroom/internal/timeutil"
	"github.com/getsentry/vroom/internal/utils"
)

func TestFlamegraphAggregation(t *testing.T) {
	tests := []struct {
		name     string
		profiles []sample.Profile
		output   speedscope.Output
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
								{},
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
								{},
								{
									ElapsedSinceStartNS: 10,
									StackID:             1,
								},
								{
									ElapsedSinceStartNS: 20,
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
						EndValue:     5,
						IsMainThread: true,
						Samples: [][]int{
							{0, 1},
							{2, 0},
							{2},
							{3},
						},
						SamplesProfiles: [][]int{
							{0},
							{1},
							{0},
							{1},
						},
						SamplesExamples:   [][]int{{}, {}, {}, {}},
						Type:              "sampled",
						Unit:              "count",
						Weights:           []uint64{1, 1, 1, 2},
						SampleCounts:      []uint64{1, 1, 1, 2},
						SampleDurationsNs: []uint64{10, 10, 10, 20},
					},
				},
				Shared: speedscope.SharedData{
					Frames: []speedscope.Frame{
						{Image: "test.package", Name: "a"},
						{Image: "test.package", Name: "b"},
						{Image: "test.package", IsApplication: true, Name: "c"},
						{Image: "test.package", Name: "e"},
					},
					ProfileIDs: []string{"ab1", "cd2"},
				},
			},
		},
	}

	options := cmp.Options{
		cmp.AllowUnexported(timeutil.Time{}),
		// This option will order profile IDs since we only want to compare values and not order.
		cmpopts.SortSlices(func(a, b string) bool {
			return a < b
		}),
		// This option will order stacks since we only want to compare values and not order.
		cmpopts.SortSlices(func(a, b []int) bool {
			al, bl := len(a), len(b)
			if al != bl {
				// Smallest slice first
				return al < bl
			}
			for i := 0; i < al; i++ {
				if a[i] != b[i] {
					// Slice with the first different smaller index first
					return a[i] < b[i]
				}
			}
			// Both slices are 0, we don't change the order
			return false
		}),
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
				addCallTreeToFlamegraph(&ft, callTrees[0], annotateWithProfileID(p.ID()))
			}

			if diff := testutil.Diff(toSpeedscope(context.TODO(), ft, 10, 99), test.output, options); diff != "" {
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
		examples  []utils.ExampleMetadata
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
					Frame:         frame.Frame{Function: "function1"},
					ProfileIDs:    make(map[string]struct{}),
					Profiles:      make(map[utils.ExampleMetadata]struct{}),
					Children: []*nodetree.Node{
						{
							DurationNS:    10_000_000,
							EndNS:         50_000_000,
							Fingerprint:   9531802423075301657,
							IsApplication: true,
							Name:          "function2",
							SampleCount:   1,
							StartNS:       40_000_000,
							Frame:         frame.Frame{Function: "function2"},
							ProfileIDs:    make(map[string]struct{}),
							Profiles:      make(map[utils.ExampleMetadata]struct{}),
						},
					},
				},
			},
			examples: []utils.ExampleMetadata{
				utils.NewExampleFromProfileID(1, "2", 10_000_000, 50_000_000),
				utils.NewExampleFromProfilerChunk(3, "4", "5", "6", &threadID, 10_000_000, 50_000_000),
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
						SamplesProfiles:   [][]int{{}, {}},
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
						{Name: "function1", IsApplication: true},
						{Name: "function2", IsApplication: true},
					},
					Profiles: []utils.ExampleMetadata{
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
		cmpopts.SortSlices(func(a, b utils.ExampleMetadata) bool {
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
