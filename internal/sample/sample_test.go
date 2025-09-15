package sample

import (
	"testing"

	"github.com/getsentry/vroom/internal/examples"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/testutil"
	"github.com/getsentry/vroom/internal/transaction"
)

func TestReplaceIdleStacks(t *testing.T) {
	tests := []struct {
		name  string
		trace Trace
		want  Trace
	}{
		{
			name: "replace idle stacks between 2 actives",
			trace: Trace{
				Samples: []Sample{
					{StackID: 1, ElapsedSinceStartNS: 10},
					{StackID: 0, ElapsedSinceStartNS: 20},
					{StackID: 0, ElapsedSinceStartNS: 30},
					{StackID: 0, ElapsedSinceStartNS: 40},
					{StackID: 2, ElapsedSinceStartNS: 50},
				},
				Stacks: []Stack{
					{},
					{4, 3, 2, 1, 0},
					{4, 2, 1, 0},
				},
				Frames: []frame.Frame{
					{Function: "function0"},
					{Function: "function1"},
					{Function: "function2"},
					{Function: "function3"},
					{Function: "function4"},
				},
			},
			want: Trace{
				Samples: []Sample{
					{StackID: 1, ElapsedSinceStartNS: 10},
					{StackID: 3, ElapsedSinceStartNS: 20, State: "idle"},
					{StackID: 3, ElapsedSinceStartNS: 30, State: "idle"},
					{StackID: 3, ElapsedSinceStartNS: 40, State: "idle"},
					{StackID: 2, ElapsedSinceStartNS: 50},
				},
				Stacks: []Stack{
					{},
					{4, 3, 2, 1, 0},
					{4, 2, 1, 0},
					{2, 1, 0},
				},
				Frames: []frame.Frame{
					{Function: "function0"},
					{Function: "function1"},
					{Function: "function2"},
					{Function: "function3"},
					{Function: "function4"},
				},
			},
		},
		{
			name: "replace idle stacks between 2 actives with idle around",
			trace: Trace{
				Samples: []Sample{
					{StackID: 0, ElapsedSinceStartNS: 10},
					{StackID: 1, ElapsedSinceStartNS: 20},
					{StackID: 0, ElapsedSinceStartNS: 30},
					{StackID: 2, ElapsedSinceStartNS: 40},
					{StackID: 0, ElapsedSinceStartNS: 50},
				},
				Stacks: []Stack{
					{},
					{4, 3, 2, 1, 0},
					{4, 2, 1, 0},
				},
				Frames: []frame.Frame{
					{Function: "function0"},
					{Function: "function1"},
					{Function: "function2"},
					{Function: "function3"},
					{Function: "function4"},
				},
			},
			want: Trace{
				Samples: []Sample{
					{StackID: 0, ElapsedSinceStartNS: 10, State: "idle"},
					{StackID: 1, ElapsedSinceStartNS: 20},
					{StackID: 3, ElapsedSinceStartNS: 30, State: "idle"},
					{StackID: 2, ElapsedSinceStartNS: 40},
					{StackID: 0, ElapsedSinceStartNS: 50, State: "idle"},
				},
				Stacks: []Stack{
					{},
					{4, 3, 2, 1, 0},
					{4, 2, 1, 0},
					{2, 1, 0},
				},
				Frames: []frame.Frame{
					{Function: "function0"},
					{Function: "function1"},
					{Function: "function2"},
					{Function: "function3"},
					{Function: "function4"},
				},
			},
		},
		{
			name: "do nothing since only one active stack",
			trace: Trace{
				Samples: []Sample{
					{StackID: 0, ElapsedSinceStartNS: 10},
					{StackID: 0, ElapsedSinceStartNS: 20},
					{StackID: 1, ElapsedSinceStartNS: 30},
					{StackID: 0, ElapsedSinceStartNS: 40},
					{StackID: 0, ElapsedSinceStartNS: 50},
				},
				Stacks: []Stack{
					{},
					{4, 3, 2, 1, 0},
				},
				Frames: []frame.Frame{
					{Function: "function0"},
					{Function: "function1"},
					{Function: "function2"},
					{Function: "function3"},
					{Function: "function4"},
				},
			},
			want: Trace{
				Samples: []Sample{
					{StackID: 0, ElapsedSinceStartNS: 10, State: "idle"},
					{StackID: 0, ElapsedSinceStartNS: 20, State: "idle"},
					{StackID: 1, ElapsedSinceStartNS: 30},
					{StackID: 0, ElapsedSinceStartNS: 40, State: "idle"},
					{StackID: 0, ElapsedSinceStartNS: 50, State: "idle"},
				},
				Stacks: []Stack{
					{},
					{4, 3, 2, 1, 0},
				},
				Frames: []frame.Frame{
					{Function: "function0"},
					{Function: "function1"},
					{Function: "function2"},
					{Function: "function3"},
					{Function: "function4"},
				},
			},
		},
		{
			name: "replace idle stacks between 2 actives on different threads",
			trace: Trace{
				Samples: []Sample{
					{StackID: 1, ElapsedSinceStartNS: 10, ThreadID: 1},
					{StackID: 1, ElapsedSinceStartNS: 10, ThreadID: 2},
					{StackID: 0, ElapsedSinceStartNS: 20, ThreadID: 1},
					{StackID: 0, ElapsedSinceStartNS: 20, ThreadID: 2},
					{StackID: 0, ElapsedSinceStartNS: 30, ThreadID: 1},
					{StackID: 0, ElapsedSinceStartNS: 30, ThreadID: 2},
					{StackID: 0, ElapsedSinceStartNS: 40, ThreadID: 1},
					{StackID: 0, ElapsedSinceStartNS: 40, ThreadID: 2},
					{StackID: 2, ElapsedSinceStartNS: 50, ThreadID: 1},
					{StackID: 2, ElapsedSinceStartNS: 50, ThreadID: 2},
				},
				Stacks: []Stack{
					{},
					{4, 3, 2, 1, 0},
					{4, 2, 1, 0},
				},
				Frames: []frame.Frame{
					{Function: "function0"},
					{Function: "function1"},
					{Function: "function2"},
					{Function: "function3"},
					{Function: "function4"},
				},
			},
			want: Trace{
				Samples: []Sample{
					{StackID: 1, ElapsedSinceStartNS: 10, ThreadID: 1},
					{StackID: 1, ElapsedSinceStartNS: 10, ThreadID: 2},
					{StackID: 3, ElapsedSinceStartNS: 20, ThreadID: 1, State: "idle"},
					{StackID: 4, ElapsedSinceStartNS: 20, ThreadID: 2, State: "idle"},
					{StackID: 3, ElapsedSinceStartNS: 30, ThreadID: 1, State: "idle"},
					{StackID: 4, ElapsedSinceStartNS: 30, ThreadID: 2, State: "idle"},
					{StackID: 3, ElapsedSinceStartNS: 40, ThreadID: 1, State: "idle"},
					{StackID: 4, ElapsedSinceStartNS: 40, ThreadID: 2, State: "idle"},
					{StackID: 2, ElapsedSinceStartNS: 50, ThreadID: 1},
					{StackID: 2, ElapsedSinceStartNS: 50, ThreadID: 2},
				},
				Stacks: []Stack{
					{},
					{4, 3, 2, 1, 0},
					{4, 2, 1, 0},
					{2, 1, 0},
					{2, 1, 0},
				},
				Frames: []frame.Frame{
					{Function: "function0"},
					{Function: "function1"},
					{Function: "function2"},
					{Function: "function3"},
					{Function: "function4"},
				},
			},
		},
		{
			name: "replace multiple idle stacks between 2 actives with idle stacks around",
			trace: Trace{
				Samples: []Sample{
					{StackID: 0, ElapsedSinceStartNS: 10},
					{StackID: 1, ElapsedSinceStartNS: 20},
					{StackID: 0, ElapsedSinceStartNS: 30},
					{StackID: 2, ElapsedSinceStartNS: 40},
					{StackID: 0, ElapsedSinceStartNS: 50},
					{StackID: 3, ElapsedSinceStartNS: 60},
					{StackID: 0, ElapsedSinceStartNS: 70},
				},
				Stacks: []Stack{
					{},
					{4, 3, 2, 1, 0},
					{4, 2, 1, 0},
					{4, 1, 0},
				},
				Frames: []frame.Frame{
					{Function: "function0"},
					{Function: "function1"},
					{Function: "function2"},
					{Function: "function3"},
					{Function: "function4"},
				},
			},
			want: Trace{
				Samples: []Sample{
					{StackID: 0, ElapsedSinceStartNS: 10, State: "idle"},
					{StackID: 1, ElapsedSinceStartNS: 20},
					{StackID: 4, ElapsedSinceStartNS: 30, State: "idle"},
					{StackID: 2, ElapsedSinceStartNS: 40},
					{StackID: 5, ElapsedSinceStartNS: 50, State: "idle"},
					{StackID: 3, ElapsedSinceStartNS: 60},
					{StackID: 0, ElapsedSinceStartNS: 70, State: "idle"},
				},
				Stacks: []Stack{
					{},
					{4, 3, 2, 1, 0},
					{4, 2, 1, 0},
					{4, 1, 0},
					{2, 1, 0},
					{1, 0},
				},
				Frames: []frame.Frame{
					{Function: "function0"},
					{Function: "function1"},
					{Function: "function2"},
					{Function: "function3"},
					{Function: "function4"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.trace.ReplaceIdleStacks()
			if diff := testutil.Diff(test.trace, test.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestInlinesProduceDifferentIDs(t *testing.T) {
	instructionAddress := "0x55bd050e168d"
	inline1 := frame.Frame{
		File:            "futures.rs",
		Function:        "symbolicator::examples.:futures::measure::{{closure}}",
		Line:            167,
		InstructionAddr: instructionAddress,
	}

	inline2 := frame.Frame{
		File:            "mod.rs",
		Function:        "\u003ccore::future::from_generator::GenFuture\u003cT\u003e as core::future::future::Future\u003e::poll",
		Line:            91,
		InstructionAddr: instructionAddress,
	}

	if inline1.ID() == inline2.ID() {
		t.Fatal("Error: 2 different inline frames have the same ID")
	}
}

func TestSameSymbolDifferentLinesProduceDifferentIDs(t *testing.T) {
	frame1 := frame.Frame{
		File:            "mod.rs",
		Function:        "test",
		Line:            95,
		InstructionAddr: "0x55bd050e168d",
		SymAddr:         "0x55bd0485d020",
	}

	frame2 := frame.Frame{
		File:            "mod.rs",
		Function:        "test",
		Line:            91,
		InstructionAddr: "0x75bf057e162f",
		SymAddr:         "0x55bd0485d020",
	}

	if frame1.ID() == frame2.ID() {
		t.Fatal("Error: 2 different frames with the same sym_address have the same ID")
	}
}

func TestIsInline(t *testing.T) {
	// symbolicated but with a sym_addr
	// so this is not an inline
	normalFrame1 := frame.Frame{
		Status:  "symbolicated",
		SymAddr: "0x55bd0485d020",
	}
	if normalFrame1.IsInline() {
		t.Fatal("normal frame classified as inline")
	}

	// non-native (python, etc.)
	normalFrame2 := frame.Frame{
		Status:  "",
		SymAddr: "",
	}
	if normalFrame2.IsInline() {
		t.Fatal("normal frame classified as inline")
	}

	inlineFrame1 := frame.Frame{
		Status:  "symbolicated",
		SymAddr: "",
	}
	if !inlineFrame1.IsInline() {
		t.Fatal("inline frame classified as normal")
	}
}

func TestCallTrees(t *testing.T) {
	tests := []struct {
		name    string
		profile Profile
		want    map[uint64][]*nodetree.Node
	}{
		{
			name: "call tree with multiple samples per frame",
			profile: Profile{
				RawProfile: RawProfile{
					Transaction: transaction.Transaction{ActiveThreadID: 1},
					Trace: Trace{
						Samples: []Sample{
							{StackID: 0, ElapsedSinceStartNS: 10, ThreadID: 1},
							{StackID: 1, ElapsedSinceStartNS: 40, ThreadID: 1},
							{StackID: 1, ElapsedSinceStartNS: 50, ThreadID: 1},
						},
						Stacks: []Stack{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: {
					{
						DurationNS:    40,
						EndNS:         50,
						Fingerprint:   15444731332182868858,
						IsApplication: true,
						Name:          "function0",
						Occurrence:    1,
						SampleCount:   2,
						StartNS:       10,
						Frame:         frame.Frame{Function: "function0"},
						ProfileIDs:    make(map[string]struct{}),
						Profiles:      make(map[examples.ExampleMetadata]struct{}),
						Children: []*nodetree.Node{
							{
								DurationNS:    40,
								EndNS:         50,
								StartNS:       10,
								Fingerprint:   14164357600995800812,
								IsApplication: true,
								Name:          "function1",
								Occurrence:    1,
								SampleCount:   2,
								Frame:         frame.Frame{Function: "function1"},
								ProfileIDs:    make(map[string]struct{}),
								Profiles:      make(map[examples.ExampleMetadata]struct{}),
								Children: []*nodetree.Node{
									{
										DurationNS:    10,
										EndNS:         50,
										Fingerprint:   9531802423075301657,
										IsApplication: true,
										Name:          "function2",
										Occurrence:    1,
										SampleCount:   1,
										StartNS:       40,
										Frame:         frame.Frame{Function: "function2"},
										ProfileIDs:    make(map[string]struct{}),
										Profiles:      make(map[examples.ExampleMetadata]struct{}),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "call tree with single sample frames",
			profile: Profile{
				RawProfile: RawProfile{
					Transaction: transaction.Transaction{ActiveThreadID: 1},
					Trace: Trace{
						Samples: []Sample{
							{StackID: 0, ElapsedSinceStartNS: 10, ThreadID: 1},
							{StackID: 1, ElapsedSinceStartNS: 40, ThreadID: 1},
						},
						Stacks: []Stack{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: {
					{
						DurationNS:    30,
						EndNS:         40,
						Fingerprint:   15444731332182868858,
						IsApplication: true,
						Name:          "function0",
						Occurrence:    1,
						SampleCount:   1,
						StartNS:       10,
						Frame:         frame.Frame{Function: "function0"},
						ProfileIDs:    make(map[string]struct{}),
						Profiles:      make(map[examples.ExampleMetadata]struct{}),
						Children: []*nodetree.Node{
							{
								DurationNS:    30,
								EndNS:         40,
								Fingerprint:   14164357600995800812,
								IsApplication: true,
								Name:          "function1",
								Occurrence:    1,
								SampleCount:   1,
								StartNS:       10,
								Frame:         frame.Frame{Function: "function1"},
								ProfileIDs:    make(map[string]struct{}),
								Profiles:      make(map[examples.ExampleMetadata]struct{}),
							},
						},
					},
				},
			},
		},
		{
			name: "call tree with single samples",
			profile: Profile{
				RawProfile: RawProfile{
					Transaction: transaction.Transaction{ActiveThreadID: 1},
					Trace: Trace{
						Samples: []Sample{
							{StackID: 0, ElapsedSinceStartNS: 10, ThreadID: 1},
							{StackID: 1, ElapsedSinceStartNS: 20, ThreadID: 1},
							{StackID: 2, ElapsedSinceStartNS: 30, ThreadID: 1},
						},
						Stacks: []Stack{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: {
					{
						DurationNS:    10,
						EndNS:         20,
						Fingerprint:   15444731332182868858,
						IsApplication: true,
						Name:          "function0",
						Occurrence:    1,
						SampleCount:   1,
						StartNS:       10,
						Frame:         frame.Frame{Function: "function0"},
						ProfileIDs:    make(map[string]struct{}),
						Profiles:      make(map[examples.ExampleMetadata]struct{}),
					},
					{
						DurationNS:    10,
						EndNS:         30,
						Fingerprint:   15444731332182868859,
						IsApplication: true,
						Name:          "function1",
						Occurrence:    1,
						SampleCount:   1,
						StartNS:       20,
						Frame:         frame.Frame{Function: "function1"},
						ProfileIDs:    make(map[string]struct{}),
						Profiles:      make(map[examples.ExampleMetadata]struct{}),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callTrees, err := test.profile.CallTrees()
			if err != nil {
				t.Fatalf("error while generating call trees: %+v\n", err)
			}
			if diff := testutil.Diff(callTrees, test.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestTrimCocoaStacks(t *testing.T) {
	tests := []struct {
		name   string
		input  Profile
		output Profile
	}{
		{
			name: "Remove frames leading to main",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.Cocoa,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function1",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function2",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "main",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "missing"},
								Platform: "cocoa",
							},
						},
						Stacks: []Stack{
							{1, 0, 2, 3, 3},
						},
					},
				},
			},
			output: Profile{
				RawProfile: RawProfile{
					Platform: platform.Cocoa,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function1",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function2",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "main",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "missing"},
								InApp:    &testutil.False,
								Platform: "cocoa",
								Status:   "missing",
							},
						},
						Stacks: []Stack{
							{1, 0, 2},
						},
					},
				},
			},
		},
		{
			name: "Remove frames in-between main and a symbolicated frame",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.Cocoa,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function1",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function2",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "main",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "missing"},
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "start_sim",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
						},
						Stacks: []Stack{
							{1, 0, 2, 3, 3, 4},
						},
					},
				},
			},
			output: Profile{
				RawProfile: RawProfile{
					Platform: platform.Cocoa,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function1",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function2",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "main",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "missing"},
								InApp:    &testutil.False,
								Platform: "cocoa",
								Status:   "missing",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "start_sim",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
						},
						Stacks: []Stack{
							{1, 0, 2, 4},
						},
					},
				},
			},
		},
		{
			name: "Remove nothing since we couldn't find main",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.Cocoa,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function1",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function2",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "missing"},
								Function: "unsymbolicated_main",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "missing"},
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "start_sim",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
						},
						Stacks: []Stack{
							{1, 0, 2, 3, 3, 4},
						},
					},
				},
			},
			output: Profile{
				RawProfile: RawProfile{
					Platform: platform.Cocoa,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function1",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function2",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "missing"},
								Function: "unsymbolicated_main",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "missing",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "missing"},
								InApp:    &testutil.False,
								Platform: "cocoa",
								Status:   "missing",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "start_sim",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
						},
						Stacks: []Stack{
							{1, 0, 2, 3, 3, 4},
						},
					},
				},
			},
		},
		{
			name: "Remove frames on many stacks",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.Cocoa,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function1",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function2",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "main",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "missing"},
								Platform: "cocoa",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "start_sim",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
						},
						Stacks: []Stack{
							{0, 2, 3, 4, 3},
							{1, 0, 2, 3, 4, 3},
							{0, 2, 3, 4, 3},
							{1, 0, 2, 3, 4, 3},
							{0, 2, 3, 4, 3},
						},
					},
				},
			},
			output: Profile{
				RawProfile: RawProfile{
					Platform: platform.Cocoa,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function1",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "function2",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "main",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "missing"},
								InApp:    &testutil.False,
								Platform: "cocoa",
								Status:   "missing",
							},
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "start_sim",
								InApp:    &testutil.True,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
						},
						Stacks: []Stack{
							{0, 2, 4},
							{1, 0, 2, 4},
							{0, 2, 4},
							{1, 0, 2, 4},
							{0, 2, 4},
						},
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

func TestTrimPythonStacks(t *testing.T) {
	tests := []struct {
		name   string
		input  Profile
		output Profile
	}{
		{
			name: "Remove module frame at the end of a stack",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.Python,
					Trace: Trace{
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
						Stacks: []Stack{
							{1, 0},
						},
					},
				},
			},
			output: Profile{
				RawProfile: RawProfile{
					Platform: platform.Python,
					Trace: Trace{
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
						Stacks: []Stack{
							{1},
						},
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

func TestNormalizeFramesPerPlatform(t *testing.T) {
	tests := []struct {
		name   string
		input  Profile
		output Profile
	}{
		{
			name: "cocoa",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.Cocoa,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "main",
								Package:  "/private/var/containers/foo",
							},
						},
						Stacks: []Stack{
							{0},
						},
					},
				},
			},
			output: Profile{
				RawProfile: RawProfile{
					Platform: platform.Cocoa,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "main",
								Package:  "/private/var/containers/foo",
								Platform: platform.Cocoa,
								InApp:    &testutil.False,
								Status:   "symbolicated",
							},
						},
						Stacks: []Stack{
							{0},
						},
					},
				},
			},
		},
		{
			name: "rust",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.Rust,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "main",
								Package:  "/usr/local/foo",
							},
						},
						Stacks: []Stack{
							{0},
						},
					},
				},
			},
			output: Profile{
				RawProfile: RawProfile{
					Platform: platform.Rust,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "main",
								Package:  "/usr/local/foo",
								Platform: platform.Rust,
								InApp:    &testutil.True,
								Status:   "symbolicated",
							},
						},
						Stacks: []Stack{
							{0},
						},
					},
				},
			},
		},
		{
			name: "python",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.Python,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Function: "Threading.run",
								File:     "threading.py",
								Module:   "threading",
								Path:     "/usr/local/lib/python3.8/threading.py",
								Platform: "python",
							},
						},
						Stacks: []Stack{
							{0},
						},
					},
				},
			},
			output: Profile{
				RawProfile: RawProfile{
					Platform: platform.Python,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Function: "Threading.run",
								File:     "threading.py",
								Module:   "threading",
								Path:     "/usr/local/lib/python3.8/threading.py",
								InApp:    &testutil.False,
								Platform: "python",
							},
						},
						Stacks: []Stack{
							{0},
						},
					},
				},
			},
		},
		{
			name: "react-native with cocoa hermes frame",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.JavaScript,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "hermes::vm::Interpreter::createObjectFromBuffer(hermes::vm::Runtime\u0026, hermes::vm::CodeBlock*, unsigned int, unsigned int, unsigned int)",
								Package:  "/private/var/containers/Bundle/Application/0DA082D7-05F5-413F-892B-642FD331230C/BIGW.app/Frameworks/hermes.framework/hermes",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
						},
						Stacks: []Stack{
							{0},
						},
					},
				},
			},
			output: Profile{
				RawProfile: RawProfile{
					Platform: platform.JavaScript,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "hermes::vm::Interpreter::createObjectFromBuffer(hermes::vm::Runtime\u0026, hermes::vm::CodeBlock*, unsigned int, unsigned int, unsigned int)",
								Package:  "/private/var/containers/Bundle/Application/0DA082D7-05F5-413F-892B-642FD331230C/BIGW.app/Frameworks/hermes.framework/hermes",
								InApp:    &testutil.False,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
						},
						Stacks: []Stack{
							{0},
						},
					},
				},
			},
		},
		{
			name: "react-native with cocoa system library",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.JavaScript,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "swift_conformsToProtocolMaybeInstantiateSuperclasses(swift::TargetMetadata\u003cswift::InProcess\u003e const*, swift::TargetProtocolDescriptor\u003cswift::InProcess\u003e const*, bool)::$_8::operator()((anonymous namespace)::ConformanceSection const\u0026) const::{lambda(swift::TargetProtocolConformanceDescriptor\u003cswift::InProcess\u003e const\u0026)#1}::operator()(swift::TargetProtocolConformanceDescriptor\u003cswift::InProcess\u003e const\u0026) const",
								Package:  "/usr/lib/swift/libswiftCore.dylib",
								InApp:    &testutil.True,
								Platform: "cocoa",
							},
						},
						Stacks: []Stack{
							{0},
						},
					},
				},
			},
			output: Profile{
				RawProfile: RawProfile{
					Platform: platform.JavaScript,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "swift_conformsToProtocolMaybeInstantiateSuperclasses(swift::TargetMetadata\u003cswift::InProcess\u003e const*, swift::TargetProtocolDescriptor\u003cswift::InProcess\u003e const*, bool)::$_8::operator()((anonymous namespace)::ConformanceSection const\u0026) const::{lambda(swift::TargetProtocolConformanceDescriptor\u003cswift::InProcess\u003e const\u0026)#1}::operator()(swift::TargetProtocolConformanceDescriptor\u003cswift::InProcess\u003e const\u0026) const",
								Package:  "/usr/lib/swift/libswiftCore.dylib",
								InApp:    &testutil.False,
								Platform: "cocoa",
								Status:   "symbolicated",
							},
						},
						Stacks: []Stack{
							{0},
						},
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

func TestCallTreesFingerprintPerPlatform(t *testing.T) {
	tests := []struct {
		name   string
		input  Profile
		output map[uint64][]*nodetree.Node
	}{
		{
			name: "cocoa",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.Cocoa,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "main",
								Package:  "/private/var/containers/foo",
							},
						},
						Stacks: []Stack{
							{0},
						},
						Samples: []Sample{
							{
								ElapsedSinceStartNS: 0,
								StackID:             0,
								ThreadID:            0,
							},
							{
								ElapsedSinceStartNS: 10,
								StackID:             0,
								ThreadID:            0,
							},
						},
					},
				},
			},
			output: map[uint64][]*nodetree.Node{
				0: {
					{
						DurationNS:    10,
						EndNS:         10,
						Fingerprint:   1628006971372193492,
						IsApplication: false,
						Name:          "main",
						Package:       "foo",
						Occurrence:    1,
						SampleCount:   1,
						ProfileIDs:    map[string]struct{}{},
						Profiles:      make(map[examples.ExampleMetadata]struct{}),
						Frame: frame.Frame{
							Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
							Function: "main",
							InApp:    &testutil.False,
							Package:  "/private/var/containers/foo",
							Platform: platform.Cocoa,
							Status:   "symbolicated",
						},
					},
				},
			},
		},
		{
			name: "rust",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.Rust,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
								Function: "main",
								Package:  "/usr/local/foo",
							},
						},
						Stacks: []Stack{
							{0},
						},
						Samples: []Sample{
							{
								ElapsedSinceStartNS: 0,
								StackID:             0,
								ThreadID:            0,
							},
							{
								ElapsedSinceStartNS: 10,
								StackID:             0,
								ThreadID:            0,
							},
						},
					},
				},
			},
			output: map[uint64][]*nodetree.Node{
				0: {
					{
						DurationNS:    10,
						EndNS:         10,
						Fingerprint:   1628006971372193492,
						IsApplication: true,
						Name:          "main",
						Package:       "foo",
						Occurrence:    1,
						SampleCount:   1,
						ProfileIDs:    map[string]struct{}{},
						Profiles:      make(map[examples.ExampleMetadata]struct{}),
						Frame: frame.Frame{
							Data:     frame.Data{SymbolicatorStatus: "symbolicated"},
							Function: "main",
							Package:  "/usr/local/foo",
							Status:   "symbolicated",
							Platform: platform.Rust,
							InApp:    &testutil.True,
						},
					},
				},
			},
		},
		{
			name: "python",
			input: Profile{
				RawProfile: RawProfile{
					Platform: platform.Python,
					Trace: Trace{
						Frames: []frame.Frame{
							{
								Function: "Threading.run",
								File:     "threading.py",
								Module:   "threading",
								Path:     "/usr/local/lib/python3.8/threading.py",
								Platform: "python",
							},
						},
						Stacks: []Stack{
							{0},
						},
						Samples: []Sample{
							{
								ElapsedSinceStartNS: 0,
								StackID:             0,
								ThreadID:            0,
							},
							{
								ElapsedSinceStartNS: 10,
								StackID:             0,
								ThreadID:            0,
							},
						},
					},
				},
			},
			output: map[uint64][]*nodetree.Node{
				0: {
					{
						DurationNS:    10,
						EndNS:         10,
						Fingerprint:   12857020554704472368,
						IsApplication: false,
						Name:          "Threading.run",
						Package:       "threading",
						Path:          "/usr/local/lib/python3.8/threading.py",
						Occurrence:    1,
						SampleCount:   1,
						ProfileIDs:    map[string]struct{}{},
						Profiles:      make(map[examples.ExampleMetadata]struct{}),
						Frame: frame.Frame{
							Function: "Threading.run",
							File:     "threading.py",
							Module:   "threading",
							InApp:    &testutil.False,
							Path:     "/usr/local/lib/python3.8/threading.py",
							Platform: "python",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.input.Normalize()
			callTrees, err := test.input.CallTrees()
			if err != nil {
				t.Fatal(err)
			}
			if diff := testutil.Diff(callTrees, test.output); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
