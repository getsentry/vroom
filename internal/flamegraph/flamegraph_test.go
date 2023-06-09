package flamegraph

import (
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/testutil"
	"github.com/getsentry/vroom/internal/timeutil"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
						EndValue:     7,
						IsMainThread: true,
						Samples: [][]int{
							{0, 1},
							{0},
							{2, 0},
							{2},
							{3},
						},
						SamplesProfiles: [][]int{
							{0, 1},
							{0},
							{1},
							{0},
							{1},
						},
						Type:              "sampled",
						Unit:              "count",
						Weights:           []uint64{3, 1, 1, 1, 1},
						SampleCounts:      []uint64{3, 1, 1, 1, 1},
						SampleDurationsNs: []uint64{20, 10, 0, 10, 10},
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
				addCallTreeToFlamegraph(&ft, callTrees[0], p.ID())
			}

			if diff := testutil.Diff(toSpeedscope(ft, 1, 99), test.output, options); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
