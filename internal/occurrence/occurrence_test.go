package occurrence

import (
	"testing"

	"github.com/getsentry/vroom/internal/frame"
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
