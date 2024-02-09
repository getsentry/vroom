package occurrence

import (
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/testutil"
)

func TestNormalizeAndroidStackTrace(t *testing.T) {
	tests := []struct {
		name   string
		input  []frame.Frame
		output []frame.Frame
	}{
		{
			name: "Normalize Android stack trace",
			input: []frame.Frame{
				{
					Package:  "com.google.gson",
					Function: "com.google.gson.JSONDecode.decode()",
				},
			},
			output: []frame.Frame{
				{
					Package:  "com.google.gson",
					Function: "JSONDecode.decode()",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizeAndroidStackTrace(tt.input)
			if diff := testutil.Diff(tt.input, tt.output); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestFromRegressedFunction(t *testing.T) {
	f := frame.Frame{
		Module:   "foo",
		Function: "bar",
	}
	tests := []struct {
		name             string
		frame            frame.Frame
		function         RegressedFunction
		expectedType     Type
		expectedTitle    IssueTitle
		expectedSubtitle string
	}{
		{
			name:  "released",
			frame: f,
			function: RegressedFunction{
				OrganizationID:  1,
				ProjectID:       1,
				ProfileID:       "",
				Fingerprint:     0,
				AggregateRange1: 100_000_000,
				AggregateRange2: 200_000_000,
			},
			expectedType:     2011,
			expectedTitle:    "Function Regression",
			expectedSubtitle: "Duration increased from 100ms to 200ms (P95).",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			occ := FromRegressedFunction(platform.Python, tt.function, tt.frame)
			if occ.Type != tt.expectedType {
				t.Fatalf("Occurrent type mismatch: got %v want %v\n", occ.Type, tt.expectedType)
			}
			if occ.IssueTitle != tt.expectedTitle {
				t.Fatalf("Occurrent title mismatch: got %v want %v\n", occ.IssueTitle, tt.expectedTitle)
			}
			if occ.Subtitle != tt.expectedSubtitle {
				t.Fatalf("Occurrent subtitle mismatch: got %v want %v\n", occ.Subtitle, tt.expectedSubtitle)
			}
		})
	}
}
