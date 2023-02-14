package flamegraph

import (
	"encoding/json"
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/speedscope"
)

var sampledProfile = sample.SampleProfile{
	Platform: "cocoa",
	Version:  "v1",
	Trace: sample.Trace{
		Frames: []frame.Frame{
			{
				Function: "a",
				Package:  "test.package",
				Path:     "/tmp",
			},
			{
				Function: "b",
				Package:  "test.package",
				Path:     "/tmp",
			},
			{
				Function: "c",
				Package:  "test.package",
				Path:     "/tmp",
			},
		}, //end frames
		Stacks: []sample.Stack{
			{1, 0}, // b,a
			{2},    // c
			{1, 0}, // b,a
			{0},    // a
		},
		Samples: []sample.Sample{
			{
				ElapsedSinceStartNS: 0,
				StackID:             0,
				ThreadID:            0,
			},
			{
				ElapsedSinceStartNS: 10,
				StackID:             1,
				ThreadID:            0,
			},
			{
				ElapsedSinceStartNS: 20,
				StackID:             2,
				ThreadID:            0,
			},
			{
				ElapsedSinceStartNS: 20,
				StackID:             3,
				ThreadID:            0,
			},
		}, // end Samples
	}, // end Trace
	Transactions: []sample.Transaction{
		{
			ActiveThreadID: 0,
		},
	}, // end Transactions
} // end prof definition

func TestProcessStacksFromCallTreesFromSampledProfile(t *testing.T) {

	bytes, err := json.Marshal(sampledProfile)
	if err != nil {
		t.Fatalf("cannot marshal sampleProfile: %V", err)
	}

	profile := profile.Profile{}
	profile.UnmarshalJSON(bytes)

	callTrees, err := profile.CallTrees()
	if err != nil {
		t.Fatalf("error trying to generate call tree: %V", err)
	}

	var stacks [][]frame.Frame
	stacksCount := make(map[uint64]int)
	ProcessStacksFromCallTrees(callTrees, &stacks, stacksCount)

	expectedStacks := [][]frame.Frame{
		{
			{
				Function: "a",
				Package:  "test.package",
			},
			{
				Function: "b",
				Package:  "test.package",
			},
		},
		{
			{
				Function: "c",
				Package:  "test.package",
			},
		},
		{
			{
				Function: "a",
				Package:  "test.package",
			},
		},
	}
	expectedCounts := []int{2, 1, 1}

	if len(stacks) != 3 {
		t.Fatalf("expected 3 stack traces, found %d", len(stacks))
	}

	for i := range stacks {
		if !sameStackTraces(stacks[i], expectedStacks[i]) || (stacksCount[stacks[i][len(stacks[i])-1].Fingerprint] != expectedCounts[i]) {
			t.Fatalf("the 2 stack traces differ: %v <> %v", stacks[i], expectedStacks[i])
		}
	}

}

func sameStackTraces(a, b []frame.Frame) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if (a[i].Function != b[i].Function) || (a[i].Package != b[i].Package) {
			return false
		}
	}
	return true
}

func TestConvertStackTracesToFlamegraph(t *testing.T) {
	bytes, err := json.Marshal(sampledProfile)
	if err != nil {
		t.Fatalf("cannot marshal sampleProfile: %V", err)
	}

	profile := profile.Profile{}
	profile.UnmarshalJSON(bytes)

	callTrees, err := profile.CallTrees()
	if err != nil {
		t.Fatalf("error trying to generate call tree: %V", err)
	}

	var stacks [][]frame.Frame
	stacksCount := make(map[uint64]int)
	// Process the CallTree twice.
	// This way we should have
	// [a, b] 4
	// [c]	  2
	// [a]    2
	ProcessStacksFromCallTrees(callTrees, &stacks, stacksCount)
	ProcessStacksFromCallTrees(callTrees, &stacks, stacksCount)

	output := ConvertStackTracesToFlamegraph(&stacks, stacksCount, 0)

	//Test frames match
	if len(output.Shared.Frames) != 3 {
		t.Fatalf("error: expected %d frames, found: %d", 3, len(output.Shared.Frames))
	} else {
		expected := []string{"a", "b", "c"}
		for i, frame := range output.Shared.Frames {
			if frame.Name != expected[i] {
				t.Fatalf("error: %d-index frame differ. Found %s, expected %s", i, frame.Name, expected[i])
			}
		}
	}

	// Test samples match
	samples := (output.Profiles[0].(speedscope.SampledProfile)).Samples
	if len(samples) != 3 {
		t.Fatalf("error: expected %d samples, found: %d", 3, len(samples))
	} else {
		expectedStacks := [][]int{
			{0, 1}, // [a, b]
			{2},    // [c]
			{0},    // [a]
		}
		expectedCounts := []int{4, 2, 2}
		for i, sample := range samples {
			if !sameSamples(sample, expectedStacks[i]) || (stacksCount[stacks[i][len(stacks[i])-1].Fingerprint] != expectedCounts[i]) {
				t.Fatalf("the %d-index samples differ. Found %v, expected %d", i, sample, expectedStacks[i])
			}
		}
	}
}

func sameSamples(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
