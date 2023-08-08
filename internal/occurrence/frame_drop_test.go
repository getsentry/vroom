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
			name: "Find longest leaf frame",
			profile: profile.New(&sample.Profile{
				RawProfile: sample.RawProfile{
					EventID: "1234567890",
					Measurements: map[string]measurements.Measurement{
						"frozen_frame_renders": {
							Unit: "nanosecond",
							Values: []measurements.MeasurementValue{
								{
									ElapsedSinceStartNs: uint64(51 * time.Millisecond),
									Value:               float64(31 * time.Millisecond),
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
								ElapsedSinceStartNS: uint64(50 * time.Millisecond),
							},
						},
					},
				},
			}),
			callTrees: map[uint64][]*nodetree.Node{
				1: {
					{
						DurationNS:    uint64(50 * time.Millisecond),
						EndNS:         uint64(50 * time.Millisecond),
						IsApplication: true,
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
								Name:          "child1-1",
								Package:       "package",
								Path:          "path",
								Frame: frame.Frame{
									Function: "child1-1",
									InApp:    &testutil.True,
									Package:  "package",
									Path:     "path",
								},
								Children: []*nodetree.Node{},
							},
							{
								DurationNS:    uint64(30 * time.Millisecond),
								EndNS:         uint64(30 * time.Millisecond),
								IsApplication: true,
								Name:          "child1-2",
								Package:       "package",
								Path:          "path",
								StartNS:       uint64(20 * time.Millisecond),
								Frame: frame.Frame{
									Function: "child1-2",
									InApp:    &testutil.True,
									Package:  "package",
									Path:     "path",
								},
								Children: []*nodetree.Node{},
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
								Function: "child1-2",
								InApp:    &testutil.True,
								Package:  "package",
								Path:     "path",
							},
						}},
						Tags: map[string]string{},
					},
					EvidenceData: map[string]interface{}{
						"frame_duration_ns":   uint64(30000000),
						"frame_module":        "",
						"frame_name":          "child1-2",
						"frame_package":       "package",
						"profile_id":          "1234567890",
						"template_name":       "profile",
						"transaction_id":      "1234",
						"transaction_name":    "some",
						"profile_duration_ns": uint64(50000000),
					},
					EvidenceDisplay: []Evidence{
						{Name: "Suspect function", Value: "child1-2", Important: true},
						{Name: "Package", Value: "package"},
					},
					Fingerprint: []string{"4ece5e7519f4f6ed944db4aeb826f1ae"},
					IssueTitle:  issueTitles[FrameDrop].IssueTitle,
					Level:       "info",
					Subtitle:    "child1-2",
					Type:        issueTitles[FrameDrop].Type,
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