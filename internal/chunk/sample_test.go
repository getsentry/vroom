package chunk

import (
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/testutil"
	"github.com/getsentry/vroom/internal/utils"
)

func TestCallTrees(t *testing.T) {
	tests := []struct {
		name  string
		chunk SampleChunk
		want  map[string][]*nodetree.Node
	}{
		{
			name: "call tree with multiple samples per frame",
			chunk: SampleChunk{
				Profile: SampleData{
					Samples: []Sample{
						{StackID: 0, Timestamp: 0.010, ThreadID: "1"},
						{StackID: 1, Timestamp: 0.040, ThreadID: "1"},
						{StackID: 1, Timestamp: 0.050, ThreadID: "1"},
					},
					Stacks: [][]int{
						{1, 0},
						{2, 1, 0},
					},
					Frames: []frame.Frame{
						{Function: "function0"},
						{Function: "function1"},
						{Function: "function2"},
					},
				},
			}, // end chunk
			want: map[string][]*nodetree.Node{
				"1": {
					{
						DurationNS:    40_000_000,
						EndNS:         50_000_000,
						Fingerprint:   15444731332182868858,
						IsApplication: true,
						Name:          "function0",
						SampleCount:   2,
						StartNS:       10_000_000,
						Frame:         frame.Frame{Function: "function0"},
						ProfileIDs:    make(map[string]struct{}),
						Profiles:      make(map[utils.ExampleMetadata]struct{}),
						Children: []*nodetree.Node{
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
					},
				},
			},
		}, // end first test
		{
			name: "call tree with single sample frames",
			chunk: SampleChunk{
				Profile: SampleData{
					Samples: []Sample{
						{StackID: 0, Timestamp: 0.010, ThreadID: "1"},
						{StackID: 1, Timestamp: 0.040, ThreadID: "1"},
					},
					Stacks: [][]int{
						{1, 0},
						{2, 1, 0},
					},
					Frames: []frame.Frame{
						{Function: "function0"},
						{Function: "function1"},
						{Function: "function2"},
					},
				},
			},
			want: map[string][]*nodetree.Node{
				"1": {
					{
						DurationNS:    30_000_000,
						EndNS:         40_000_000,
						Fingerprint:   15444731332182868858,
						IsApplication: true,
						Name:          "function0",
						SampleCount:   1,
						StartNS:       10_000_000,
						Frame:         frame.Frame{Function: "function0"},
						ProfileIDs:    make(map[string]struct{}),
						Profiles:      make(map[utils.ExampleMetadata]struct{}),
						Children: []*nodetree.Node{
							{
								DurationNS:    30_000_000,
								EndNS:         40_000_000,
								Fingerprint:   14164357600995800812,
								IsApplication: true,
								Name:          "function1",
								SampleCount:   1,
								StartNS:       10_000_000,
								Frame:         frame.Frame{Function: "function1"},
								ProfileIDs:    make(map[string]struct{}),
								Profiles:      make(map[utils.ExampleMetadata]struct{}),
							},
						},
					},
				},
			},
		}, // end first test
		{
			name: "call tree with single samples",
			chunk: SampleChunk{
				Profile: SampleData{
					Samples: []Sample{
						{StackID: 0, Timestamp: 0.010, ThreadID: "1"},
						{StackID: 1, Timestamp: 0.020, ThreadID: "1"},
						{StackID: 2, Timestamp: 0.030, ThreadID: "1"},
					},
					Stacks: [][]int{
						{0},
						{1},
						{2},
					},
					Frames: []frame.Frame{
						{Function: "function0"},
						{Function: "function1"},
						{Function: "function2"},
					},
				},
			},
			want: map[string][]*nodetree.Node{
				"1": {
					{
						DurationNS:    10_000_000,
						EndNS:         20_000_000,
						Fingerprint:   15444731332182868858,
						IsApplication: true,
						Name:          "function0",
						SampleCount:   1,
						StartNS:       10_000_000,
						Frame:         frame.Frame{Function: "function0"},
						ProfileIDs:    make(map[string]struct{}),
						Profiles:      make(map[utils.ExampleMetadata]struct{}),
					},
					{
						DurationNS:    10_000_000,
						EndNS:         30_000_000,
						Fingerprint:   15444731332182868859,
						IsApplication: true,
						Name:          "function1",
						SampleCount:   1,
						StartNS:       20_000_000,
						Frame:         frame.Frame{Function: "function1"},
						ProfileIDs:    make(map[string]struct{}),
						Profiles:      make(map[utils.ExampleMetadata]struct{}),
					},
				},
			},
		}, // end third test
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callTrees, err := test.chunk.CallTrees(nil)
			if err != nil {
				t.Fatalf("error while generating call trees: %+v\n", err)
			}
			if diff := testutil.Diff(callTrees, test.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestTrimPythonStacks(t *testing.T) {
	tests := []struct {
		name   string
		input  SampleChunk
		output SampleChunk
	}{
		{
			name: "Remove module frame at the end of a stack",
			input: SampleChunk{
				Platform: platform.Python,
				Profile: SampleData{
					Frames: []frame.Frame{
						{
							File:     "<string>",
							Module:   "__main__",
							InApp:    &testutil.True,
							Line:     11,
							Function: "<module>",
							Path:     "/usr/src/app/<string>",
							Platform: "python",
						},
						{
							File:     "app/util.py",
							Module:   "app.util",
							InApp:    &testutil.True,
							Line:     98,
							Function: "foobar",
							Path:     "/usr/src/app/util.py",
							Platform: "python",
						},
					},
					Stacks: [][]int{
						{1, 0},
					},
				},
			},
			output: SampleChunk{
				Platform: platform.Python,
				Profile: SampleData{
					Frames: []frame.Frame{
						{
							File:     "<string>",
							Module:   "__main__",
							InApp:    &testutil.True,
							Line:     11,
							Function: "<module>",
							Path:     "/usr/src/app/<string>",
							Platform: "python",
						},
						{
							File:     "app/util.py",
							Module:   "app.util",
							InApp:    &testutil.True,
							Line:     98,
							Function: "foobar",
							Path:     "/usr/src/app/util.py",
							Platform: "python",
						},
					},
					Stacks: [][]int{
						{1},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.input.Normalize()
			if diff := testutil.Diff(test.input, test.output); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
