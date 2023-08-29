package occurrence

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/measurements"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/testutil"
	"github.com/getsentry/vroom/internal/transaction"
)

func TestFindFrameDrop(t *testing.T) {
	tests := []struct {
		name      string
		callTrees map[uint64][]*nodetree.Node
		profile   profile.Profile
		want      []*Occurrence
	}{
		{
			name: "Find a basic cause of a frame drop",
			profile: profile.New(&sample.Profile{
				RawProfile: sample.RawProfile{
					EventID: "1234567890",
					Measurements: map[string]measurements.Measurement{
						"frozen_frame_renders": {
							Unit: "nanosecond",
							Values: []measurements.MeasurementValue{
								{
									ElapsedSinceStartNs: uint64(500 * time.Millisecond),
									Value:               float64(300 * time.Millisecond),
								},
							},
						},
					},
					Transaction: transaction.Transaction{
						ActiveThreadID: 1,
						ID:             "1234",
						Name:           "some",
					},
					Platform: platform.Cocoa,
					Trace: sample.Trace{
						Samples: []sample.Sample{
							{
								ElapsedSinceStartNS: 0,
							},
							{
								ElapsedSinceStartNS: uint64(500 * time.Millisecond),
							},
						},
					},
				},
			}),
			callTrees: map[uint64][]*nodetree.Node{
				1: {
					{
						DurationNS:    uint64(500 * time.Millisecond),
						EndNS:         uint64(500 * time.Millisecond),
						IsApplication: false,
						Name:          "root",
						Package:       "package",
						Path:          "path",
						Frame: frame.Frame{
							Function: "root",
							InApp:    &testutil.False,
							Package:  "package",
							Path:     "path",
						},
						Children: []*nodetree.Node{
							{
								DurationNS:    uint64(200 * time.Millisecond),
								EndNS:         uint64(200 * time.Millisecond),
								IsApplication: true,
								Name:          "child1",
								Package:       "package",
								Path:          "path",
								Frame: frame.Frame{
									Function: "child1",
									InApp:    &testutil.True,
									Package:  "package",
									Path:     "path",
								},
							},
							{
								DurationNS:    uint64(100 * time.Millisecond),
								EndNS:         uint64(300 * time.Millisecond),
								IsApplication: true,
								Name:          "child2",
								Package:       "package",
								Path:          "path",
								StartNS:       uint64(200 * time.Millisecond),
								Frame: frame.Frame{
									Function: "child2",
									InApp:    &testutil.True,
									Package:  "package",
									Path:     "path",
								},
								Children: []*nodetree.Node{
									{
										DurationNS:    uint64(50 * time.Millisecond),
										EndNS:         uint64(250 * time.Millisecond),
										IsApplication: true,
										Name:          "child2-1",
										Package:       "package",
										Path:          "path",
										StartNS:       uint64(200 * time.Millisecond),
										Frame: frame.Frame{
											Function: "child2-1",
											InApp:    &testutil.False,
											Package:  "package",
											Path:     "path",
										},
										Children: []*nodetree.Node{},
									},
								},
							},
						},
					},
				},
			},
			want: []*Occurrence{
				{
					Culprit: "some",
					Event: Event{
						Platform: "cocoa",
						StackTrace: StackTrace{Frames: []frame.Frame{
							{
								Function: "root",
								InApp:    &testutil.False,
								Package:  "package",
								Path:     "path",
							},
							{
								Function: "child2",
								InApp:    &testutil.True,
								Package:  "package",
								Path:     "path",
							},
						}},
						Tags: map[string]string{},
					},
					EvidenceData: map[string]interface{}{
						"frame_duration_ns":   uint64(100000000),
						"frame_module":        "",
						"frame_name":          "child2",
						"frame_package":       "package",
						"profile_id":          "1234567890",
						"template_name":       "profile",
						"transaction_id":      "1234",
						"transaction_name":    "some",
						"profile_duration_ns": uint64(500000000),
					},
					EvidenceDisplay: []Evidence{
						{Name: "Suspect function", Value: "child2", Important: true},
						{Name: "Package", Value: "package"},
					},
					Fingerprint: []string{"8f2e4eaab20fd0a0acb48bb9bdd11b21"},
					IssueTitle:  issueTitles[FrameDrop].IssueTitle,
					Level:       "info",
					Subtitle:    "child2",
					Type:        issueTitles[FrameDrop].Type,
				},
			},
		},
		{
			name: "Find a deeper frame than expected",
			profile: profile.New(&sample.Profile{
				RawProfile: sample.RawProfile{
					EventID: "1234567890",
					Measurements: map[string]measurements.Measurement{
						"frozen_frame_renders": {
							Unit: "nanosecond",
							Values: []measurements.MeasurementValue{
								{
									ElapsedSinceStartNs: uint64(500 * time.Millisecond),
									Value:               float64(300 * time.Millisecond),
								},
							},
						},
					},
					Transaction: transaction.Transaction{
						ActiveThreadID: 1,
						ID:             "1234",
						Name:           "some",
					},
					Platform: platform.Cocoa,
					Trace: sample.Trace{
						Samples: []sample.Sample{
							{
								ElapsedSinceStartNS: 0,
							},
							{
								ElapsedSinceStartNS: uint64(500 * time.Millisecond),
							},
						},
					},
				},
			}),
			callTrees: map[uint64][]*nodetree.Node{
				1: {
					{
						DurationNS:    uint64(500 * time.Millisecond),
						EndNS:         uint64(500 * time.Millisecond),
						IsApplication: false,
						Name:          "root",
						Package:       "package",
						Path:          "path",
						Frame: frame.Frame{
							Function: "root",
							InApp:    &testutil.False,
							Package:  "package",
							Path:     "path",
						},
						Children: []*nodetree.Node{
							{
								DurationNS:    uint64(20 * time.Millisecond),
								EndNS:         uint64(20 * time.Millisecond),
								IsApplication: true,
								Name:          "child1",
								Package:       "package",
								Path:          "path",
								Frame: frame.Frame{
									Function: "child1-1",
									InApp:    &testutil.True,
									Package:  "package",
									Path:     "path",
								},
							},
							{
								DurationNS:    uint64(100 * time.Millisecond),
								EndNS:         uint64(300 * time.Millisecond),
								IsApplication: true,
								Name:          "child2",
								Package:       "package",
								Path:          "path",
								StartNS:       uint64(200 * time.Millisecond),
								Frame: frame.Frame{
									Function: "child2",
									InApp:    &testutil.True,
									Package:  "package",
									Path:     "path",
								},
								Children: []*nodetree.Node{
									{
										DurationNS:    uint64(100 * time.Millisecond),
										EndNS:         uint64(300 * time.Millisecond),
										IsApplication: false,
										Name:          "child2-1",
										Package:       "package",
										Path:          "path",
										StartNS:       uint64(200 * time.Millisecond),
										Frame: frame.Frame{
											Function: "child2-1",
											InApp:    &testutil.False,
											Package:  "package",
											Path:     "path",
										},
										Children: []*nodetree.Node{
											{
												DurationNS:    uint64(100 * time.Millisecond),
												EndNS:         uint64(300 * time.Millisecond),
												IsApplication: true,
												Name:          "child2-1-1",
												Package:       "package",
												Path:          "path",
												StartNS:       uint64(200 * time.Millisecond),
												Frame: frame.Frame{
													Function: "child2-1-1",
													InApp:    &testutil.True,
													Package:  "package",
													Path:     "path",
												},
												Children: []*nodetree.Node{
													{
														DurationNS: uint64(
															100 * time.Millisecond,
														),
														EndNS: uint64(
															300 * time.Millisecond,
														),
														IsApplication: false,
														Name:          "child2-1-1-1",
														Package:       "package",
														Path:          "path",
														StartNS: uint64(
															200 * time.Millisecond,
														),
														Frame: frame.Frame{
															Function: "child2-1-1-1",
															InApp:    &testutil.False,
															Package:  "package",
															Path:     "path",
														},
														Children: []*nodetree.Node{
															{
																DurationNS: uint64(
																	100 * time.Millisecond,
																),
																EndNS: uint64(
																	300 * time.Millisecond,
																),
																IsApplication: false,
																Name:          "child2-1-1-1-1",
																Package:       "package",
																Path:          "path",
																StartNS: uint64(
																	200 * time.Millisecond,
																),
																Frame: frame.Frame{
																	Function: "child2-1-1-1-1",
																	InApp:    &testutil.False,
																	Package:  "package",
																	Path:     "path",
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []*Occurrence{
				{
					Culprit: "some",
					Event: Event{
						Platform: "cocoa",
						StackTrace: StackTrace{Frames: []frame.Frame{
							{
								Function: "root",
								InApp:    &testutil.False,
								Package:  "package",
								Path:     "path",
							},
							{
								Function: "child2",
								InApp:    &testutil.True,
								Package:  "package",
								Path:     "path",
							},
							{
								Function: "child2-1",
								InApp:    &testutil.False,
								Package:  "package",
								Path:     "path",
							},
							{
								Function: "child2-1-1",
								InApp:    &testutil.True,
								Package:  "package",
								Path:     "path",
							},
						}},
						Tags: map[string]string{},
					},
					EvidenceData: map[string]interface{}{
						"frame_duration_ns":   uint64(100000000),
						"frame_module":        "",
						"frame_name":          "child2-1-1",
						"frame_package":       "package",
						"profile_id":          "1234567890",
						"template_name":       "profile",
						"transaction_id":      "1234",
						"transaction_name":    "some",
						"profile_duration_ns": uint64(500000000),
					},
					EvidenceDisplay: []Evidence{
						{
							Name:      "Suspect function",
							Value:     "child2-1-1",
							Important: true,
						},
						{Name: "Package", Value: "package"},
					},
					Fingerprint: []string{
						"5f5584b142aab31b5f33b26884cdec56",
					},
					IssueTitle: issueTitles[FrameDrop].IssueTitle,
					Level:      "info",
					Subtitle:   "child2-1-1",
					Type:       issueTitles[FrameDrop].Type,
				},
			},
		},
		{
			name: "Find a deeper and longer frame in shorter parent system frame",
			profile: profile.New(&sample.Profile{
				RawProfile: sample.RawProfile{
					EventID: "1234567890",
					Measurements: map[string]measurements.Measurement{
						"frozen_frame_renders": {
							Unit: "nanosecond",
							Values: []measurements.MeasurementValue{
								{
									ElapsedSinceStartNs: uint64(500 * time.Millisecond),
									Value:               float64(450 * time.Millisecond),
								},
							},
						},
					},
					Transaction: transaction.Transaction{
						ActiveThreadID: 1,
						ID:             "1234",
						Name:           "some",
					},
					Platform: platform.Cocoa,
					Trace: sample.Trace{
						Samples: []sample.Sample{
							{
								ElapsedSinceStartNS: 0,
							},
							{
								ElapsedSinceStartNS: uint64(500 * time.Millisecond),
							},
						},
					},
				},
			}),
			callTrees: map[uint64][]*nodetree.Node{
				1: {
					{
						DurationNS:    uint64(500 * time.Millisecond),
						EndNS:         uint64(500 * time.Millisecond),
						IsApplication: false,
						Name:          "root",
						Package:       "package",
						Path:          "path",
						Frame: frame.Frame{
							Function: "root",
							InApp:    &testutil.False,
							Package:  "package",
							Path:     "path",
						},
						Children: []*nodetree.Node{
							{
								DurationNS:    uint64(50 * time.Millisecond),
								EndNS:         uint64(50 * time.Millisecond),
								IsApplication: true,
								Name:          "child1",
								Package:       "package",
								Path:          "path",
								Frame: frame.Frame{
									Function: "child1",
									InApp:    &testutil.True,
									Package:  "package",
									Path:     "path",
								},
								Children: []*nodetree.Node{
									{
										DurationNS:    uint64(50 * time.Millisecond),
										EndNS:         uint64(50 * time.Millisecond),
										IsApplication: true,
										Name:          "child1-1",
										Package:       "package",
										Path:          "path",
										Frame: frame.Frame{
											Function: "child1-1",
											InApp:    &testutil.True,
											Package:  "package",
											Path:     "path",
										},
									},
								},
							},
							{
								DurationNS:    uint64(200 * time.Millisecond),
								EndNS:         uint64(250 * time.Millisecond),
								IsApplication: false,
								Name:          "child2",
								Package:       "package",
								Path:          "path",
								StartNS:       uint64(50 * time.Millisecond),
								Frame: frame.Frame{
									Function: "child2",
									InApp:    &testutil.False,
									Package:  "package",
									Path:     "path",
								},
								Children: []*nodetree.Node{
									{
										DurationNS:    uint64(100 * time.Millisecond),
										EndNS:         uint64(250 * time.Millisecond),
										IsApplication: true,
										Name:          "child2-1",
										Package:       "package",
										Path:          "path",
										StartNS:       uint64(150 * time.Millisecond),
										Frame: frame.Frame{
											Function: "child2-1",
											InApp:    &testutil.True,
											Package:  "package",
											Path:     "path",
										},
									},
								},
							},
							{
								DurationNS:    uint64(250 * time.Millisecond),
								EndNS:         uint64(500 * time.Millisecond),
								IsApplication: false,
								Name:          "child3",
								Package:       "package",
								Path:          "path",
								StartNS:       uint64(250 * time.Millisecond),
								Frame: frame.Frame{
									Function: "child3",
									InApp:    &testutil.False,
									Package:  "package",
									Path:     "path",
								},
								Children: []*nodetree.Node{
									{
										DurationNS:    uint64(50 * time.Millisecond),
										EndNS:         uint64(300 * time.Millisecond),
										IsApplication: true,
										Name:          "child3-1",
										Package:       "package",
										Path:          "path",
										StartNS:       uint64(250 * time.Millisecond),
										Frame: frame.Frame{
											Function: "child3-1",
											InApp:    &testutil.True,
											Package:  "package",
											Path:     "path",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []*Occurrence{
				{
					Culprit: "some",
					Event: Event{
						Platform: "cocoa",
						StackTrace: StackTrace{Frames: []frame.Frame{
							{
								Function: "root",
								InApp:    &testutil.False,
								Package:  "package",
								Path:     "path",
							},
							{
								Function: "child2",
								InApp:    &testutil.False,
								Package:  "package",
								Path:     "path",
							},
							{
								Function: "child2-1",
								InApp:    &testutil.True,
								Package:  "package",
								Path:     "path",
							},
						}},
						Tags: map[string]string{},
					},
					EvidenceData: map[string]interface{}{
						"frame_duration_ns":   uint64(100000000),
						"frame_module":        "",
						"frame_name":          "child2-1",
						"frame_package":       "package",
						"profile_id":          "1234567890",
						"template_name":       "profile",
						"transaction_id":      "1234",
						"transaction_name":    "some",
						"profile_duration_ns": uint64(500000000),
					},
					EvidenceDisplay: []Evidence{
						{
							Name:      "Suspect function",
							Value:     "child2-1",
							Important: true,
						},
						{Name: "Package", Value: "package"},
					},
					Fingerprint: []string{
						"1247dbde8be089556f3ae631f81e73c2",
					},
					IssueTitle: issueTitles[FrameDrop].IssueTitle,
					Level:      "info",
					Subtitle:   "child2-1",
					Type:       issueTitles[FrameDrop].Type,
				},
			},
		},
		{
			name: "Make sure we're biased towards earlier frames",
			profile: profile.New(&sample.Profile{
				RawProfile: sample.RawProfile{
					EventID: "1234567890",
					Measurements: map[string]measurements.Measurement{
						"frozen_frame_renders": {
							Unit: "nanosecond",
							Values: []measurements.MeasurementValue{
								{
									ElapsedSinceStartNs: uint64(500 * time.Millisecond),
									Value:               float64(500 * time.Millisecond),
								},
							},
						},
					},
					Transaction: transaction.Transaction{
						ActiveThreadID: 1,
						ID:             "1234",
						Name:           "some",
					},
					Platform: platform.Cocoa,
					Trace: sample.Trace{
						Samples: []sample.Sample{
							{
								ElapsedSinceStartNS: 0,
							},
							{
								ElapsedSinceStartNS: uint64(500 * time.Millisecond),
							},
						},
					},
				},
			}),
			callTrees: map[uint64][]*nodetree.Node{
				1: {
					{
						DurationNS:    uint64(500 * time.Millisecond),
						EndNS:         uint64(500 * time.Millisecond),
						IsApplication: false,
						Name:          "root",
						Package:       "package",
						Path:          "path",
						Frame: frame.Frame{
							Function: "root",
							InApp:    &testutil.False,
							Package:  "package",
							Path:     "path",
						},
						Children: []*nodetree.Node{
							{
								DurationNS:    uint64(50 * time.Millisecond),
								EndNS:         uint64(50 * time.Millisecond),
								IsApplication: true,
								Name:          "child1",
								Package:       "package",
								Path:          "path",
								Frame: frame.Frame{
									Function: "child1",
									InApp:    &testutil.True,
									Package:  "package",
									Path:     "path",
								},
							},
							{
								DurationNS:    uint64(200 * time.Millisecond),
								EndNS:         uint64(300 * time.Millisecond),
								IsApplication: false,
								Name:          "child2",
								Package:       "package",
								Path:          "path",
								StartNS:       uint64(100 * time.Millisecond),
								Frame: frame.Frame{
									Function: "child2",
									InApp:    &testutil.False,
									Package:  "package",
									Path:     "path",
								},
								Children: []*nodetree.Node{
									{
										DurationNS:    uint64(200 * time.Millisecond),
										EndNS:         uint64(300 * time.Millisecond),
										IsApplication: true,
										Name:          "child2-1",
										Package:       "package",
										Path:          "path",
										StartNS:       uint64(100 * time.Millisecond),
										Frame: frame.Frame{
											Function: "child2-1",
											InApp:    &testutil.True,
											Package:  "package",
											Path:     "path",
										},
									},
								},
							},
							{
								DurationNS:    uint64(200 * time.Millisecond),
								EndNS:         uint64(500 * time.Millisecond),
								IsApplication: true,
								Name:          "child3",
								Package:       "package",
								Path:          "path",
								StartNS:       uint64(300 * time.Millisecond),
								Frame: frame.Frame{
									Function: "child3",
									InApp:    &testutil.True,
									Package:  "package",
									Path:     "path",
								},
							},
						},
					},
				},
			},
			want: []*Occurrence{
				{
					Culprit: "some",
					Event: Event{
						Platform: "cocoa",
						StackTrace: StackTrace{Frames: []frame.Frame{
							{
								Function: "root",
								InApp:    &testutil.False,
								Package:  "package",
								Path:     "path",
							},
							{
								Function: "child2",
								InApp:    &testutil.False,
								Package:  "package",
								Path:     "path",
							},
							{
								Function: "child2-1",
								InApp:    &testutil.True,
								Package:  "package",
								Path:     "path",
							},
						}},
						Tags: map[string]string{},
					},
					EvidenceData: map[string]interface{}{
						"frame_duration_ns":   uint64(200000000),
						"frame_module":        "",
						"frame_name":          "child2-1",
						"frame_package":       "package",
						"profile_id":          "1234567890",
						"template_name":       "profile",
						"transaction_id":      "1234",
						"transaction_name":    "some",
						"profile_duration_ns": uint64(500000000),
					},
					EvidenceDisplay: []Evidence{
						{
							Name:      "Suspect function",
							Value:     "child2-1",
							Important: true,
						},
						{Name: "Package", Value: "package"},
					},
					Fingerprint: []string{
						"1247dbde8be089556f3ae631f81e73c2",
					},
					IssueTitle: issueTitles[FrameDrop].IssueTitle,
					Level:      "info",
					Subtitle:   "child2-1",
					Type:       issueTitles[FrameDrop].Type,
				},
			},
		},
	}

	options := cmp.Options{
		cmpopts.IgnoreFields(Event{}, "ID"),
		cmpopts.IgnoreFields(Occurrence{}, "DetectionTime", "ID"),
		cmpopts.IgnoreUnexported(Occurrence{}),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var occurrences []*Occurrence
			findFrameDropCause(tt.profile, tt.callTrees, &occurrences)
			if diff := testutil.Diff(occurrences, tt.want, options); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
