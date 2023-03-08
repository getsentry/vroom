package sample

import (
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
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
		Function:        "symbolicator::utils::futures::measure::{{closure}}",
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
						DurationNS:    50,
						EndNS:         50,
						Fingerprint:   15444731332182868858,
						IsApplication: true,
						Name:          "function0",
						SampleCount:   3,
						Frame:         frame.Frame{Function: "function0"},
						ProfileIDs:    make(map[string]struct{}),
						Children: []*nodetree.Node{
							{
								DurationNS:    50,
								EndNS:         50,
								Fingerprint:   14164357600995800812,
								IsApplication: true,
								Name:          "function1",
								SampleCount:   3,
								Frame:         frame.Frame{Function: "function1"},
								ProfileIDs:    make(map[string]struct{}),
								Children: []*nodetree.Node{
									{
										DurationNS:    40,
										EndNS:         50,
										Fingerprint:   9531802423075301657,
										IsApplication: true,
										Name:          "function2",
										SampleCount:   2,
										StartNS:       10,
										Frame:         frame.Frame{Function: "function2"},
										ProfileIDs:    make(map[string]struct{}),
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
						DurationNS:    40,
						EndNS:         40,
						Fingerprint:   15444731332182868858,
						IsApplication: true,
						Name:          "function0",
						SampleCount:   2,
						Frame:         frame.Frame{Function: "function0"},
						ProfileIDs:    make(map[string]struct{}),
						Children: []*nodetree.Node{
							{
								DurationNS:    40,
								EndNS:         40,
								Fingerprint:   14164357600995800812,
								IsApplication: true,
								Name:          "function1",
								SampleCount:   2,
								Frame:         frame.Frame{Function: "function1"},
								ProfileIDs:    make(map[string]struct{}),
								Children: []*nodetree.Node{
									{
										DurationNS:    30,
										EndNS:         40,
										Fingerprint:   9531802423075301657,
										IsApplication: true,
										Name:          "function2",
										SampleCount:   1,
										StartNS:       10,
										Frame:         frame.Frame{Function: "function2"},
										ProfileIDs:    make(map[string]struct{}),
									},
								},
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
						EndNS:         10,
						Fingerprint:   15444731332182868858,
						IsApplication: true,
						Name:          "function0",
						SampleCount:   1,
						Frame:         frame.Frame{Function: "function0"},
						ProfileIDs:    make(map[string]struct{}),
					},
					{
						DurationNS:    10,
						EndNS:         20,
						Fingerprint:   15444731332182868859,
						IsApplication: true,
						Name:          "function1",
						SampleCount:   1,
						StartNS:       10,
						Frame:         frame.Frame{Function: "function1"},
						ProfileIDs:    make(map[string]struct{}),
					},
					{
						DurationNS:    10,
						EndNS:         30,
						Fingerprint:   15444731332182868856,
						IsApplication: true,
						Name:          "function2",
						SampleCount:   1,
						StartNS:       20,
						Frame:         frame.Frame{Function: "function2"},
						ProfileIDs:    make(map[string]struct{}),
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
