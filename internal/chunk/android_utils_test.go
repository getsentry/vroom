package chunk

import (
	"testing"
	"time"

	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/testutil"
)

var androidChunk1 = AndroidChunk{
	Timestamp:  0.0,
	DurationNS: 1500,
	ID:         "1a009sd87",
	Platform:   platform.Android,
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
							Secs:  0,
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
							Secs:  0,
							Nanos: 1500,
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
							Secs:  0,
							Nanos: 2000,
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
							Secs:  0,
							Nanos: 2500,
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
		},
		StartTime: 0,
		Threads: []profile.AndroidThread{
			{
				ID:   1,
				Name: "main",
			},
		},
	},
}

var androidChunk2 = AndroidChunk{
	Timestamp:  2.5e-6,
	DurationNS: 2000,
	ID:         "ee3409d8",
	Platform:   platform.Android,
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
							Secs:  0,
							Nanos: 500,
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
							Secs:  0,
							Nanos: 1000,
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
							Secs:  0,
							Nanos: 1500,
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
							Secs:  0,
							Nanos: 2000,
						},
					},
				},
			},
		},
		Methods: []profile.AndroidMethod{
			{
				ClassName: "class4",
				ID:        1,
				Name:      "method4",
				Signature: "()",
			},
			{
				ClassName: "class1",
				ID:        2,
				Name:      "method1",
				Signature: "()",
			},
			{
				ClassName: "class2",
				ID:        3,
				Name:      "method2",
				Signature: "()",
			},
		},
		StartTime: 0,
		Threads: []profile.AndroidThread{
			{
				ID:   1,
				Name: "main",
			},
		},
	},
}

func TestSpeedscopeFromAndroidChunks(t *testing.T) {
	tests := []struct {
		name  string
		have  []AndroidChunk
		want  speedscope.Output
		start uint64
		end   uint64
	}{
		{
			name: "All chunks included in the time range",
			have: []AndroidChunk{androidChunk1, androidChunk2},
			want: speedscope.Output{
				AndroidClock: "Dual",
				DurationNS:   4500,
				ChunkID:      "1a009sd87",
				Platform:     platform.Android,
				Profiles: []any{
					&speedscope.EventedProfile{
						EndValue: 4500,
						Events: []speedscope.Event{
							{
								Type:  "O",
								Frame: 0,
								At:    1000,
							},
							{
								Type:  "O",
								Frame: 1,
								At:    1500,
							},
							{
								Type:  "O",
								Frame: 2,
								At:    2000,
							},
							{
								Type:  "C",
								Frame: 2,
								At:    2500,
							},
							{
								Type:  "O",
								Frame: 3,
								At:    3000,
							},
							{
								Type:  "C",
								Frame: 3,
								At:    3500,
							},
							{
								Type:  "C",
								Frame: 1,
								At:    4000,
							},
							{
								Type:  "C",
								Frame: 0,
								At:    4500,
							},
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
						{Image: "class3", IsApplication: true, Name: "class3.method3()"},
						{Image: "class4", IsApplication: true, Name: "class4.method4()"},
					},
				},
				Metadata: speedscope.ProfileMetadata{
					ProfileView: speedscope.ProfileView{
						Timestamp: time.Unix(0, 0).UTC(),
					},
				},
			},
			start: 0,
			end:   6000,
		},
		{
			name: "First chunk begins before allowed range (overlap )",
			have: []AndroidChunk{androidChunk1, androidChunk2},
			want: speedscope.Output{
				AndroidClock: "Dual",
				DurationNS:   3000,
				ChunkID:      "1a009sd87",
				Platform:     platform.Android,
				Profiles: []any{
					&speedscope.EventedProfile{
						EndValue: 3000,
						Events: []speedscope.Event{
							{
								Type:  "O",
								Frame: 1,
								At:    0,
							},
							{
								Type:  "O",
								Frame: 2,
								At:    500,
							},
							{
								Type:  "C",
								Frame: 2,
								At:    1000,
							},
							{
								Type:  "O",
								Frame: 3,
								At:    1500,
							},
							{
								Type:  "C",
								Frame: 3,
								At:    2000,
							},
							{
								Type:  "C",
								Frame: 1,
								At:    2500,
							},
						},
						Name:       "main",
						StartValue: 0,
						ThreadID:   1,
						Type:       "evented",
						Unit:       "nanoseconds",
					},
				},
				Shared: speedscope.SharedData{
					Frames: []speedscope.Frame{
						{Image: "class1", IsApplication: true, Name: "class1.method1()"},
						{Image: "class2", IsApplication: true, Name: "class2.method2()"},
						{Image: "class3", IsApplication: true, Name: "class3.method3()"},
						{Image: "class4", IsApplication: true, Name: "class4.method4()"},
					},
				},
				Metadata: speedscope.ProfileMetadata{
					ProfileView: speedscope.ProfileView{
						Timestamp: time.Unix(0, 1500).UTC(),
					},
				},
			},
			start: 1500,
			end:   6000,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := SpeedscopeFromAndroidChunks(test.have, test.start, test.end)
			if err != nil {
				t.Fatal(err)
			}
			if diff := testutil.Diff(s, test.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
