package sample

import (
	"testing"

	"github.com/getsentry/vroom/internal/testutil"
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
					Sample{StackID: 1, ElapsedSinceStartNS: 10},
					Sample{StackID: 0, ElapsedSinceStartNS: 20},
					Sample{StackID: 0, ElapsedSinceStartNS: 30},
					Sample{StackID: 0, ElapsedSinceStartNS: 40},
					Sample{StackID: 2, ElapsedSinceStartNS: 50},
				},
				Stacks: []Stack{
					Stack{},
					Stack{4, 3, 2, 1, 0},
					Stack{4, 2, 1, 0},
				},
				Frames: []Frame{
					Frame{Function: "function0"},
					Frame{Function: "function1"},
					Frame{Function: "function2"},
					Frame{Function: "function3"},
					Frame{Function: "function4"},
				},
			},
			want: Trace{
				Samples: []Sample{
					Sample{StackID: 1, ElapsedSinceStartNS: 10},
					Sample{StackID: 3, ElapsedSinceStartNS: 20, State: "idle"},
					Sample{StackID: 3, ElapsedSinceStartNS: 30, State: "idle"},
					Sample{StackID: 3, ElapsedSinceStartNS: 40, State: "idle"},
					Sample{StackID: 2, ElapsedSinceStartNS: 50},
				},
				Stacks: []Stack{
					Stack{},
					Stack{4, 3, 2, 1, 0},
					Stack{4, 2, 1, 0},
					Stack{2, 1, 0},
				},
				Frames: []Frame{
					Frame{Function: "function0"},
					Frame{Function: "function1"},
					Frame{Function: "function2"},
					Frame{Function: "function3"},
					Frame{Function: "function4"},
				},
			},
		},
		{
			name: "replace idle stacks between 2 actives with idle around",
			trace: Trace{
				Samples: []Sample{
					Sample{StackID: 0, ElapsedSinceStartNS: 10},
					Sample{StackID: 1, ElapsedSinceStartNS: 20},
					Sample{StackID: 0, ElapsedSinceStartNS: 30},
					Sample{StackID: 2, ElapsedSinceStartNS: 40},
					Sample{StackID: 0, ElapsedSinceStartNS: 50},
				},
				Stacks: []Stack{
					Stack{},
					Stack{4, 3, 2, 1, 0},
					Stack{4, 2, 1, 0},
				},
				Frames: []Frame{
					Frame{Function: "function0"},
					Frame{Function: "function1"},
					Frame{Function: "function2"},
					Frame{Function: "function3"},
					Frame{Function: "function4"},
				},
			},
			want: Trace{
				Samples: []Sample{
					Sample{StackID: 0, ElapsedSinceStartNS: 10, State: "idle"},
					Sample{StackID: 1, ElapsedSinceStartNS: 20},
					Sample{StackID: 3, ElapsedSinceStartNS: 30, State: "idle"},
					Sample{StackID: 2, ElapsedSinceStartNS: 40},
					Sample{StackID: 0, ElapsedSinceStartNS: 50, State: "idle"},
				},
				Stacks: []Stack{
					Stack{},
					Stack{4, 3, 2, 1, 0},
					Stack{4, 2, 1, 0},
					Stack{2, 1, 0},
				},
				Frames: []Frame{
					Frame{Function: "function0"},
					Frame{Function: "function1"},
					Frame{Function: "function2"},
					Frame{Function: "function3"},
					Frame{Function: "function4"},
				},
			},
		},
		{
			name: "do nothing since only one active stack",
			trace: Trace{
				Samples: []Sample{
					Sample{StackID: 0, ElapsedSinceStartNS: 10},
					Sample{StackID: 0, ElapsedSinceStartNS: 20},
					Sample{StackID: 1, ElapsedSinceStartNS: 30},
					Sample{StackID: 0, ElapsedSinceStartNS: 40},
					Sample{StackID: 0, ElapsedSinceStartNS: 50},
				},
				Stacks: []Stack{
					Stack{},
					Stack{4, 3, 2, 1, 0},
				},
				Frames: []Frame{
					Frame{Function: "function0"},
					Frame{Function: "function1"},
					Frame{Function: "function2"},
					Frame{Function: "function3"},
					Frame{Function: "function4"},
				},
			},
			want: Trace{
				Samples: []Sample{
					Sample{StackID: 0, ElapsedSinceStartNS: 10, State: "idle"},
					Sample{StackID: 0, ElapsedSinceStartNS: 20, State: "idle"},
					Sample{StackID: 1, ElapsedSinceStartNS: 30},
					Sample{StackID: 0, ElapsedSinceStartNS: 40, State: "idle"},
					Sample{StackID: 0, ElapsedSinceStartNS: 50, State: "idle"},
				},
				Stacks: []Stack{
					Stack{},
					Stack{4, 3, 2, 1, 0},
				},
				Frames: []Frame{
					Frame{Function: "function0"},
					Frame{Function: "function1"},
					Frame{Function: "function2"},
					Frame{Function: "function3"},
					Frame{Function: "function4"},
				},
			},
		},
		{
			name: "replace idle stacks between 2 actives on different threads",
			trace: Trace{
				Samples: []Sample{
					Sample{StackID: 1, ElapsedSinceStartNS: 10, ThreadID: 1},
					Sample{StackID: 1, ElapsedSinceStartNS: 10, ThreadID: 2},
					Sample{StackID: 0, ElapsedSinceStartNS: 20, ThreadID: 1},
					Sample{StackID: 0, ElapsedSinceStartNS: 20, ThreadID: 2},
					Sample{StackID: 0, ElapsedSinceStartNS: 30, ThreadID: 1},
					Sample{StackID: 0, ElapsedSinceStartNS: 30, ThreadID: 2},
					Sample{StackID: 0, ElapsedSinceStartNS: 40, ThreadID: 1},
					Sample{StackID: 0, ElapsedSinceStartNS: 40, ThreadID: 2},
					Sample{StackID: 2, ElapsedSinceStartNS: 50, ThreadID: 1},
					Sample{StackID: 2, ElapsedSinceStartNS: 50, ThreadID: 2},
				},
				Stacks: []Stack{
					Stack{},
					Stack{4, 3, 2, 1, 0},
					Stack{4, 2, 1, 0},
				},
				Frames: []Frame{
					Frame{Function: "function0"},
					Frame{Function: "function1"},
					Frame{Function: "function2"},
					Frame{Function: "function3"},
					Frame{Function: "function4"},
				},
			},
			want: Trace{
				Samples: []Sample{
					Sample{StackID: 1, ElapsedSinceStartNS: 10, ThreadID: 1},
					Sample{StackID: 1, ElapsedSinceStartNS: 10, ThreadID: 2},
					Sample{StackID: 3, ElapsedSinceStartNS: 20, ThreadID: 1, State: "idle"},
					Sample{StackID: 4, ElapsedSinceStartNS: 20, ThreadID: 2, State: "idle"},
					Sample{StackID: 3, ElapsedSinceStartNS: 30, ThreadID: 1, State: "idle"},
					Sample{StackID: 4, ElapsedSinceStartNS: 30, ThreadID: 2, State: "idle"},
					Sample{StackID: 3, ElapsedSinceStartNS: 40, ThreadID: 1, State: "idle"},
					Sample{StackID: 4, ElapsedSinceStartNS: 40, ThreadID: 2, State: "idle"},
					Sample{StackID: 2, ElapsedSinceStartNS: 50, ThreadID: 1},
					Sample{StackID: 2, ElapsedSinceStartNS: 50, ThreadID: 2},
				},
				Stacks: []Stack{
					Stack{},
					Stack{4, 3, 2, 1, 0},
					Stack{4, 2, 1, 0},
					Stack{2, 1, 0},
					Stack{2, 1, 0},
				},
				Frames: []Frame{
					Frame{Function: "function0"},
					Frame{Function: "function1"},
					Frame{Function: "function2"},
					Frame{Function: "function3"},
					Frame{Function: "function4"},
				},
			},
		},
		{
			name: "replace multiple idle stacks between 2 actives with idle stacks around",
			trace: Trace{
				Samples: []Sample{
					Sample{StackID: 0, ElapsedSinceStartNS: 10},
					Sample{StackID: 1, ElapsedSinceStartNS: 20},
					Sample{StackID: 0, ElapsedSinceStartNS: 30},
					Sample{StackID: 2, ElapsedSinceStartNS: 40},
					Sample{StackID: 0, ElapsedSinceStartNS: 50},
					Sample{StackID: 3, ElapsedSinceStartNS: 60},
					Sample{StackID: 0, ElapsedSinceStartNS: 70},
				},
				Stacks: []Stack{
					Stack{},
					Stack{4, 3, 2, 1, 0},
					Stack{4, 2, 1, 0},
					Stack{4, 1, 0},
				},
				Frames: []Frame{
					Frame{Function: "function0"},
					Frame{Function: "function1"},
					Frame{Function: "function2"},
					Frame{Function: "function3"},
					Frame{Function: "function4"},
				},
			},
			want: Trace{
				Samples: []Sample{
					Sample{StackID: 0, ElapsedSinceStartNS: 10, State: "idle"},
					Sample{StackID: 1, ElapsedSinceStartNS: 20},
					Sample{StackID: 4, ElapsedSinceStartNS: 30, State: "idle"},
					Sample{StackID: 2, ElapsedSinceStartNS: 40},
					Sample{StackID: 5, ElapsedSinceStartNS: 50, State: "idle"},
					Sample{StackID: 3, ElapsedSinceStartNS: 60},
					Sample{StackID: 0, ElapsedSinceStartNS: 70, State: "idle"},
				},
				Stacks: []Stack{
					Stack{},
					Stack{4, 3, 2, 1, 0},
					Stack{4, 2, 1, 0},
					Stack{4, 1, 0},
					Stack{2, 1, 0},
					Stack{1, 0},
				},
				Frames: []Frame{
					Frame{Function: "function0"},
					Frame{Function: "function1"},
					Frame{Function: "function2"},
					Frame{Function: "function3"},
					Frame{Function: "function4"},
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
	instruction_address := "0x55bd050e168d"
	inline_1 := Frame{
		File:            "futures.rs",
		Function:        "symbolicator::utils::futures::measure::{{closure}}",
		Line:            167,
		InstructionAddr: instruction_address,
	}

	inline_2 := Frame{
		File:            "mod.rs",
		Function:        "\u003ccore::future::from_generator::GenFuture\u003cT\u003e as core::future::future::Future\u003e::poll",
		Line:            91,
		InstructionAddr: instruction_address,
	}

	if inline_1.ID() == inline_2.ID() {
		t.Fatal("Error: 2 different inline frames have the same ID")
	}
}

func TestSameSymbolDifferentLinesProduceDifferentIDs(t *testing.T) {
	frame_1 := Frame{
		File:            "mod.rs",
		Function:        "test",
		Line:            95,
		InstructionAddr: "0x55bd050e168d",
		SymAddr:         "0x55bd0485d020",
	}

	frame_2 := Frame{
		File:            "mod.rs",
		Function:        "test",
		Line:            91,
		InstructionAddr: "0x75bf057e162f",
		SymAddr:         "0x55bd0485d020",
	}

	if frame_1.ID() == frame_2.ID() {
		t.Fatal("Error: 2 different frames with the same sym_address have the same ID")
	}
}

func TestIsInline(t *testing.T) {
	// symbolicated but with a sym_addr
	// so this is not an inline
	normal_frame_1 := Frame{
		Status:  "symbolicated",
		SymAddr: "0x55bd0485d020",
	}
	if normal_frame_1.IsInline() {
		t.Fatal("normal frame classified as inline")
	}

	// non-native (python, etc.)
	normal_frame_2 := Frame{
		Status:  "",
		SymAddr: "",
	}
	if normal_frame_2.IsInline() {
		t.Fatal("normal frame classified as inline")
	}

	inline_frame_1 := Frame{
		Status:  "symbolicated",
		SymAddr: "",
	}
	if !inline_frame_1.IsInline() {
		t.Fatal("inline frame classified as normal")
	}
}
