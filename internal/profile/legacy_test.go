package profile

import (
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/testutil"
)

func TestSampleToAndroidFormat(t *testing.T) {
	tests := []struct {
		name   string
		input  sample.Trace
		output Android
	}{
		{
			name: "Convert Sample Profile to Android profile",
			input: sample.Trace{
				Frames: []frame.Frame{
					{Function: "a", InApp: &testutil.False},
					{Function: "b", InApp: &testutil.True},
					{Function: "c", InApp: &testutil.False},
					{Function: "d", InApp: &testutil.False},
				},
				Samples: []sample.Sample{
					{
						ElapsedSinceStartNS: 0,
						StackID:             0,
						ThreadID:            1,
					},
					{
						ElapsedSinceStartNS: 1e7,
						StackID:             1,
						ThreadID:            1,
					},
				},
				Stacks: []sample.Stack{
					{2, 1, 0},
					{3, 1, 0},
				},
				ThreadMetadata: map[string]sample.ThreadMetadata{
					"1": {
						Name: "JavaScriptThread",
					},
				},
			},
			output: Android{
				Clock: DualClock,
				Events: []AndroidEvent{
					{
						Action:   "Enter",
						ThreadID: 2,
						MethodID: 1,
						Time:     EventTime{},
					},
					{
						Action:   "Enter",
						ThreadID: 2,
						MethodID: 2,
						Time:     EventTime{},
					},
					{
						Action:   "Enter",
						ThreadID: 2,
						MethodID: 3,
						Time:     EventTime{},
					},
					{
						Action:   "Exit",
						ThreadID: 2,
						MethodID: 3,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Nanos: 1e7,
									Secs:  0,
								},
							},
						},
					},
					{
						Action:   "Enter",
						ThreadID: 2,
						MethodID: 4,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Nanos: 1e7,
									Secs:  0,
								},
							},
						},
					},
					{
						Action:   "Exit",
						ThreadID: 2,
						MethodID: 4,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Nanos: 1e7 * 2,
									Secs:  0,
								},
							},
						},
					},
					{
						Action:   "Exit",
						ThreadID: 2,
						MethodID: 2,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Nanos: 1e7 * 2,
									Secs:  0,
								},
							},
						},
					},
					{
						Action:   "Exit",
						ThreadID: 2,
						MethodID: 1,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Nanos: 1e7 * 2,
									Secs:  0,
								},
							},
						},
					},
				}, // end events
				Methods: []AndroidMethod{
					{
						ID:    1,
						Name:  "a",
						InApp: &testutil.False,
					},
					{
						ID:    2,
						Name:  "b",
						InApp: &testutil.True,
					},
					{
						ID:    3,
						Name:  "c",
						InApp: &testutil.False,
					},
					{
						ID:    4,
						Name:  "d",
						InApp: &testutil.False,
					},
				},

				Threads: []AndroidThread{
					{
						ID:   2,
						Name: "main",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			convertedProfile := sampleToAndroidFormat(tests[0].input, 1, map[uint64]void{1: {}})

			if diff := testutil.Diff(convertedProfile, test.output); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
