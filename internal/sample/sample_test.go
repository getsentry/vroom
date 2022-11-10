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
