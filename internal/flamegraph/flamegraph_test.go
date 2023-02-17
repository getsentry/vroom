package flamegraph

import (
	"encoding/json"
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/testutil"
)

var firstSampledProfile = sample.SampleProfile{
	Platform: "cocoa",
	Version:  "v1",
	Trace: sample.Trace{
		Frames: []frame.Frame{
			{
				Function: "a",
				Package:  "test.package",
			},
			{
				Function: "b",
				Package:  "test.package",
			},
			{
				Function: "c",
				Package:  "test.package",
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
				ElapsedSinceStartNS: 30,
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

var secondSampledProfile = sample.SampleProfile{
	Platform: "cocoa",
	Version:  "v1",
	Trace: sample.Trace{
		Frames: []frame.Frame{
			{
				Function: "a",
				Package:  "test.package",
			},
			{
				Function: "c",
				Package:  "test.package",
			},
			{
				Function: "e",
				Package:  "test.package",
			},
		}, //end frames
		Stacks: []sample.Stack{
			{0, 1}, // a,c
			{2},    // e
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
		}, // end Samples
	}, // end Trace
	Transactions: []sample.Transaction{
		{
			ActiveThreadID: 0,
		},
	}, // end Transactions
} // end prof definition

func TestFlamegraphSpeedscopeGeneration(t *testing.T) {
	var flamegraphTree []*nodetree.Node

	bytes, err := json.Marshal(firstSampledProfile)
	if err != nil {
		t.Fatalf("cannot marshal sampleProfile: %V", err)
	}

	var pr profile.Profile
	err = json.Unmarshal(bytes, &pr)
	if err != nil {
		t.Fatalf("error trying to unmarshal the profile: %V", err)
	}

	callTrees, err := pr.CallTrees()
	if err != nil {
		t.Fatalf("error trying to generate call tree: %V", err)
	}
	addCallTreeToFlamegraph(&flamegraphTree, callTrees[0])

	// second
	bytes, err = json.Marshal(secondSampledProfile)
	if err != nil {
		t.Fatalf("cannot marshal sampleProfile: %V", err)
	}

	err = json.Unmarshal(bytes, &pr)
	if err != nil {
		t.Fatalf("error trying to unmarshal the profile: %V", err)
	}

	callTrees, err = pr.CallTrees()
	if err != nil {
		t.Fatalf("error trying to generate call tree: %V", err)
	}

	addCallTreeToFlamegraph(&flamegraphTree, callTrees[0])

	sp := toSpeedscope(flamegraphTree, 1)
	prof := sp.Profiles[0].(speedscope.SampledProfile)

	expectedSamples := [][]int{
		{0, 1},
		{0},
		{2, 0},
		{2},
		{3},
	}

	expectedWeights := []uint64{2, 1, 1, 1, 1}

	if diff := testutil.Diff(expectedSamples, prof.Samples); diff != "" {
		t.Fatalf("expected \"%v\" but was \"%v\"", expectedSamples, prof.Samples)
	}

	if diff := testutil.Diff(expectedWeights, prof.Weights); diff != "" {
		t.Fatalf("expected \"%v\" but was \"%v\"", expectedWeights, prof.Weights)
	}
}
