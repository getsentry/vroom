package chunk

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/testutil"
)

var (
	missingExitEventsTrace = AndroidChunk{
		Profile: profile.Android{
			Clock: "Dual",
			Events: []profile.AndroidEvent{
				{
					Action:   "Enter",
					ThreadID: 1,
					MethodID: 1,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Nanos: 1000,
							},
						},
					},
				},
				{
					Action:   "Enter",
					ThreadID: 1,
					MethodID: 2,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Nanos: 1000,
							},
						},
					},
				},
				{
					Action:   "Exit",
					ThreadID: 1,
					MethodID: 1,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Nanos: 2000,
							},
						},
					},
				},
				{
					Action:   "Enter",
					ThreadID: 1,
					MethodID: 1,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Nanos: 3000,
							},
						},
					},
				},
			},
			Methods: []profile.AndroidMethod{
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
			Threads: []profile.AndroidThread{
				{
					ID:   1,
					Name: "main",
				},
			},
		},
	}
	missingEnterEventsTrace = AndroidChunk{
		Profile: profile.Android{
			Clock: "Dual",
			Events: []profile.AndroidEvent{
				{
					Action:   "Enter",
					ThreadID: 1,
					MethodID: 1,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Nanos: 1000,
							},
						},
					},
				},
				{
					Action:   "Enter",
					ThreadID: 1,
					MethodID: 3,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Nanos: 1500,
							},
						},
					},
				},
				{
					Action:   "Enter",
					ThreadID: 1,
					MethodID: 4,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Nanos: 1750,
							},
						},
					},
				},
				{
					Action:   "Exit",
					ThreadID: 1,
					MethodID: 2,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Nanos: 2000,
							},
						},
					},
				},
				{
					Action:   "Exit",
					ThreadID: 1,
					MethodID: 4,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Nanos: 2250,
							},
						},
					},
				},
				{
					Action:   "Exit",
					ThreadID: 1,
					MethodID: 3,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Nanos: 2500,
							},
						},
					},
				},
				{
					Action:   "Exit",
					ThreadID: 1,
					MethodID: 1,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Nanos: 3000,
							},
						},
					},
				},
			},
			Methods: []profile.AndroidMethod{
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
			Threads: []profile.AndroidThread{
				{
					ID:   1,
					Name: "main",
				},
			},
		},
	}
)

func TestCallTreesAndroid(t *testing.T) {
	tests := []struct {
		name  string
		chunk AndroidChunk
		tid   string
		want  map[string][]*nodetree.Node
	}{
		{
			name:  "Build call trees with missing exit events",
			chunk: missingExitEventsTrace,
			tid:   "1",
			want: map[string][]*nodetree.Node{
				"1": {
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
		},
		{
			name:  "Build call trees with missing enter events",
			chunk: missingEnterEventsTrace,
			tid:   "1",
			want: map[string][]*nodetree.Node{
				"1": {
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
		},
	}

	options := cmp.Options{
		cmpopts.IgnoreFields(nodetree.Node{}, "Fingerprint", "ProfileIDs", "Profiles"),
		cmpopts.IgnoreFields(frame.Frame{}, "File"),
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callTrees, err := test.chunk.CallTrees(&test.tid)
			if err != nil {
				t.Fatal(err)
			}
			if diff := testutil.Diff(callTrees, test.want, options); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
