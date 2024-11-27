package profile

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/testutil"
)

var (
	missingEnterEventsTrace = Android{
		Clock: "Dual",
		Events: []AndroidEvent{
			{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 1,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 1000,
						},
					},
				},
			},
			{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 3,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 1500,
						},
					},
				},
			},
			{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 4,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 1750,
						},
					},
				},
			},
			{
				Action:   "Exit",
				ThreadID: 1,
				MethodID: 2,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 2000,
						},
					},
				},
			},
			{
				Action:   "Exit",
				ThreadID: 1,
				MethodID: 4,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 2250,
						},
					},
				},
			},
			{
				Action:   "Exit",
				ThreadID: 1,
				MethodID: 3,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 2500,
						},
					},
				},
			},
			{
				Action:   "Exit",
				ThreadID: 1,
				MethodID: 1,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 3000,
						},
					},
				},
			},
		},
		Methods: []AndroidMethod{
			{
				ClassName: "class1",
				ID:        1,
				Name:      "method1",
				Signature: "()",
			},
			{
				ClassName: "class2",
				ID:        2,
				Name:      "method2",
				Signature: "()",
			},
			{
				ClassName: "class3",
				ID:        3,
				Name:      "method3",
				Signature: "()",
			},
			{
				ClassName: "class4",
				ID:        4,
				Name:      "method4",
				Signature: "()",
			},
		},
		StartTime: 398635355383000,
		Threads: []AndroidThread{
			{
				ID:   1,
				Name: "main",
			},
		},
	}
	missingExitEventsTrace = Android{
		Clock: "Dual",
		Events: []AndroidEvent{
			{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 1,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 1000,
						},
					},
				},
			},
			{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 2,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 1000,
						},
					},
				},
			},
			{
				Action:   "Exit",
				ThreadID: 1,
				MethodID: 1,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 2000,
						},
					},
				},
			},
			{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 1,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 3000,
						},
					},
				},
			},
		},
		Methods: []AndroidMethod{
			{
				ClassName: "class1",
				ID:        1,
				Name:      "method1",
				Signature: "()",
			},
			{
				ClassName: "class2",
				ID:        2,
				Name:      "method2",
				Signature: "()",
			},
		},
		StartTime: 398635355383000,
		Threads: []AndroidThread{
			{
				ID:   1,
				Name: "main",
			},
		},
	}

	nonMonotonicTrace = Android{
		Clock: "Dual",
		Events: []AndroidEvent{
			{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 1,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Secs:  1,
							Nanos: 1000,
						},
					},
				},
			},
			{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 2,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Secs:  2,
							Nanos: 1000,
						},
					},
				},
			},
			{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 3,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Secs:  7,
							Nanos: 2000,
						},
					},
				},
			},
			{
				Action:   "Exit",
				ThreadID: 1,
				MethodID: 3,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Secs:  6,
							Nanos: 3000,
						},
					},
				},
			},
			{
				Action:   "Exit",
				ThreadID: 1,
				MethodID: 2,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Secs:  6,
							Nanos: 3000,
						},
					},
				},
			},
			{
				Action:   "Exit",
				ThreadID: 1,
				MethodID: 1,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Secs:  9,
							Nanos: 3000,
						},
					},
				},
			},
			{
				Action:   "Enter",
				ThreadID: 2,
				MethodID: 1,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Secs:  1,
							Nanos: 3000,
						},
					},
				},
			},
			{
				Action:   "Enter",
				ThreadID: 2,
				MethodID: 2,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Secs:  2,
							Nanos: 3000,
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
							Secs:  2,
							Nanos: 3000,
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
							Secs:  3,
							Nanos: 3000,
						},
					},
				},
			},
		},
		StartTime: 398635355383000,
		Threads: []AndroidThread{
			{
				ID:   1,
				Name: "main",
			},
			{
				ID:   2,
				Name: "background",
			},
		},
	}
	stackDepth3EventsTrace = Android{
		Clock: "Dual",
		Events: []AndroidEvent{
			{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 1,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 1000,
						},
					},
				},
			},
			{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 3,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 1000,
						},
					},
				},
			},
			{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 4,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 1000,
						},
					},
				},
			},
			{
				Action:   "Exit",
				ThreadID: 1,
				MethodID: 2,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 2000,
						},
					},
				},
			},
			{
				Action:   "Exit",
				ThreadID: 1,
				MethodID: 4,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 2000,
						},
					},
				},
			},
			{
				Action:   "Exit",
				ThreadID: 1,
				MethodID: 3,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Nanos: 2000,
						},
					},
				},
			},
		},
		Methods: []AndroidMethod{
			{
				ClassName: "class1",
				ID:        1,
				Name:      "method1",
				Signature: "()",
			},
			{
				ClassName: "class2",
				ID:        2,
				Name:      "method2",
				Signature: "()",
			},
			{
				ClassName: "class3",
				ID:        3,
				Name:      "method3",
				Signature: "()",
			},
			{
				ClassName: "class4",
				ID:        4,
				Name:      "method4",
				Signature: "()",
			},
		},
		StartTime: 398635355383000,
		Threads: []AndroidThread{
			{
				ID:   1,
				Name: "main",
			},
		},
	}
)

func TestSpeedscope(t *testing.T) {
	tests := []struct {
		name  string
		trace Android
		want  speedscope.Output
	}{
		{
			name:  "Build speedscope with missing exit events",
			trace: missingExitEventsTrace,
			want: speedscope.Output{
				AndroidClock: "Dual",
				Profiles: []any{
					&speedscope.EventedProfile{
						EndValue: 3000,
						Events: []speedscope.Event{
							{Type: "O", Frame: 0, At: 1000},
							{Type: "O", Frame: 1, At: 1000},
							{Type: "C", Frame: 1, At: 2000},
							{Type: "C", Frame: 0, At: 2000},
							{Type: "O", Frame: 0, At: 3000},
							{Type: "C", Frame: 0, At: 3000},
						},
						Name:       "main",
						StartValue: 1000,
						ThreadID:   1,
						Type:       "evented",
						Unit:       "nanoseconds",
					},
				},
				Shared: speedscope.SharedData{
					Frames: []speedscope.Frame{
						{Image: "class1", IsApplication: true, Name: "class1.method1()"},
						{Image: "class2", IsApplication: true, Name: "class2.method2()"},
					},
				},
			},
		},
		{
			name:  "Build speedscope with missing enter events",
			trace: missingEnterEventsTrace,
			want: speedscope.Output{
				AndroidClock: "Dual",
				Profiles: []any{
					&speedscope.EventedProfile{
						EndValue: 3000,
						Events: []speedscope.Event{
							{Type: "O", Frame: 0, At: 1000},
							{Type: "O", Frame: 2, At: 1500},
							{Type: "O", Frame: 3, At: 1750},
							{Type: "C", Frame: 3, At: 2250},
							{Type: "C", Frame: 2, At: 2500},
							{Type: "C", Frame: 0, At: 3000},
						},
						Name:       "main",
						StartValue: 1000,
						ThreadID:   1,
						Type:       "evented",
						Unit:       "nanoseconds",
					},
				},
				Shared: speedscope.SharedData{
					Frames: []speedscope.Frame{
						{Image: "class1", Name: "class1.method1()", IsApplication: true},
						{Image: "class2", Name: "class2.method2()", IsApplication: true},
						{Image: "class3", Name: "class3.method3()", IsApplication: true},
						{Image: "class4", Name: "class4.method4()", IsApplication: true},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := test.trace.Speedscope()
			if err != nil {
				t.Fatalf("couldn't generate speedscope format: %+v", err)
			}
			if diff := testutil.Diff(output, test.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestCallTrees(t *testing.T) {
	tests := []struct {
		name     string
		trace    Android
		want     map[uint64][]*nodetree.Node
		maxDepth int
	}{
		{
			name:  "Build call trees with missing exit events",
			trace: missingExitEventsTrace,
			want: map[uint64][]*nodetree.Node{
				1: {
					{
						DurationNS:    1000,
						IsApplication: true,
						EndNS:         2000,
						StartNS:       1000,
						Name:          "class1.method1()",
						Package:       "class1",
						SampleCount:   1,
						Frame: frame.Frame{
							Function: "class1.method1()",
							InApp:    &testutil.True,
							MethodID: 1,
							Package:  "class1",
						},
						Children: []*nodetree.Node{
							{
								DurationNS:    1000,
								IsApplication: true,
								Name:          "class2.method2()",
								Package:       "class2",
								EndNS:         2000,
								StartNS:       1000,
								SampleCount:   1,
								Frame: frame.Frame{
									Function: "class2.method2()",
									InApp:    &testutil.True,
									MethodID: 2,
									Package:  "class2",
								},
							},
						},
					},
					{
						DurationNS:    0,
						IsApplication: true,
						Name:          "class1.method1()",
						Package:       "class1",
						EndNS:         3000,
						StartNS:       3000,
						Frame: frame.Frame{
							Function: "class1.method1()",
							MethodID: 1,
							InApp:    &testutil.True,
							Package:  "class1",
						},
					},
				},
			},
			maxDepth: MaxStackDepth,
		},
		{
			name:  "Build call trees with missing enter events",
			trace: missingEnterEventsTrace,
			want: map[uint64][]*nodetree.Node{
				1: {
					{
						DurationNS:    2000,
						IsApplication: true,
						EndNS:         3000,
						SampleCount:   1,
						StartNS:       1000,
						Package:       "class1",
						Name:          "class1.method1()",
						Frame: frame.Frame{
							Function: "class1.method1()",
							InApp:    &testutil.True,
							MethodID: 1,
							Package:  "class1",
						},
						Children: []*nodetree.Node{
							{
								DurationNS:    1000,
								IsApplication: true,
								EndNS:         2500,
								SampleCount:   1,
								StartNS:       1500,
								Package:       "class3",
								Name:          "class3.method3()",
								Frame: frame.Frame{
									Function: "class3.method3()",
									InApp:    &testutil.True,
									MethodID: 3,
									Package:  "class3",
								},
								Children: []*nodetree.Node{
									{
										DurationNS:    500,
										IsApplication: true,
										EndNS:         2250,
										SampleCount:   1,
										StartNS:       1750,
										Package:       "class4",
										Name:          "class4.method4()",
										Frame: frame.Frame{
											Function: "class4.method4()",
											InApp:    &testutil.True,
											MethodID: 4,
											Package:  "class4",
										},
									},
								},
							},
						},
					},
				},
			},
			maxDepth: MaxStackDepth,
		},
		{
			name:  "Build call trees but truncate stack depth",
			trace: stackDepth3EventsTrace,
			want: map[uint64][]*nodetree.Node{
				1: {
					{
						DurationNS:    1000,
						IsApplication: true,
						EndNS:         2000,
						StartNS:       1000,
						Name:          "class1.method1()",
						Package:       "class1",
						SampleCount:   1,
						Frame: frame.Frame{
							Function: "class1.method1()",
							InApp:    &testutil.True,
							MethodID: 1,
							Package:  "class1",
						},
					},
				},
			},
			maxDepth: 1,
		},
	}

	options := cmp.Options{
		cmpopts.IgnoreFields(nodetree.Node{}, "Fingerprint", "ProfileIDs", "Profiles"),
		cmpopts.IgnoreFields(frame.Frame{}, "File"),
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if diff := testutil.Diff(test.trace.CallTreesWithMaxDepth(test.maxDepth), test.want, options); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestFixSamplesTime(t *testing.T) {
	tests := []struct {
		name  string
		trace Android
		want  Android
	}{
		{
			name:  "Make sample secs monotonic",
			trace: nonMonotonicTrace,
			want: Android{
				Clock: "Dual",
				Events: []AndroidEvent{
					{
						Action:   "Enter",
						ThreadID: 1,
						MethodID: 1,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Secs:  1,
									Nanos: 1000,
								},
							},
						},
					},
					{
						Action:   "Enter",
						ThreadID: 1,
						MethodID: 2,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Secs:  2,
									Nanos: 1000,
								},
							},
						},
					},
					{
						Action:   "Enter",
						ThreadID: 1,
						MethodID: 3,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Secs:  7,
									Nanos: 2000,
								},
							},
						},
					},
					{
						Action:   "Exit",
						ThreadID: 1,
						MethodID: 3,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Secs:  8,
									Nanos: 2000,
								},
							},
						},
					},
					{
						Action:   "Exit",
						ThreadID: 1,
						MethodID: 2,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Secs:  8,
									Nanos: 2000,
								},
							},
						},
					},
					{
						Action:   "Exit",
						ThreadID: 1,
						MethodID: 1,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Secs:  11,
									Nanos: 2000,
								},
							},
						},
					},
					{
						Action:   "Enter",
						ThreadID: 2,
						MethodID: 1,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Secs:  1,
									Nanos: 3000,
								},
							},
						},
					},
					{
						Action:   "Enter",
						ThreadID: 2,
						MethodID: 2,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Secs:  2,
									Nanos: 3000,
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
									Secs:  2,
									Nanos: 3000,
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
									Secs:  3,
									Nanos: 3000,
								},
							},
						},
					},
				},
				StartTime: 398635355383000,
				Threads: []AndroidThread{
					{
						ID:   1,
						Name: "main",
					},
					{
						ID:   2,
						Name: "background",
					},
				},
			},
		},
	} // end tests

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.trace.FixSamplesTime()
			if diff := testutil.Diff(test.trace, test.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestAddTimeDelta(t *testing.T) {
	tests := []struct {
		name  string
		delta int64
		trace Android
		want  AndroidEvent
	}{
		{
			name:  "Delta increase seconds",
			delta: 50,
			trace: Android{
				Clock: "Dual",
				Events: []AndroidEvent{
					{
						Action:   "Enter",
						ThreadID: 1,
						MethodID: 1,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Secs:  1,
									Nanos: 1e9,
								},
							},
						},
					},
				},
				StartTime: 0,
			},
			want: AndroidEvent{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 1,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Secs:  2,
							Nanos: 50,
						},
					},
				},
			},
		},
		{
			name:  "Delta decrease nanos",
			delta: -50,
			trace: Android{
				Clock: "Dual",
				Events: []AndroidEvent{
					{
						Action:   "Enter",
						ThreadID: 1,
						MethodID: 1,
						Time: EventTime{
							Monotonic: EventMonotonic{
								Wall: Duration{
									Secs:  1,
									Nanos: 100,
								},
							},
						},
					},
				},
				StartTime: 0,
			},
			want: AndroidEvent{
				Action:   "Enter",
				ThreadID: 1,
				MethodID: 1,
				Time: EventTime{
					Monotonic: EventMonotonic{
						Wall: Duration{
							Secs:  1,
							Nanos: 50,
						},
					},
				},
			},
		},
	} // end tests

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addTimeDelta := test.trace.AddTimeDelta(test.delta)
			event := test.trace.Events[0]
			err := addTimeDelta(&event)
			if err != nil {
				t.Fatal(err)
			}
			if diff := testutil.Diff(event, test.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
