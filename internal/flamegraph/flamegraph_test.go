package flamegraph

import (
	"encoding/json"
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/testutil"
	"github.com/getsentry/vroom/internal/transaction"
)

var firstSampledProfile = sample.Profile{
	RawProfile: sample.RawProfile{
		EventID:  "ab1",
		Platform: platform.Cocoa,
		Version:  "1",
		Trace: sample.Trace{
			Frames: []frame.Frame{
				{
					Function: "a",
					Package:  "test.package",
					InApp: testutil.BoolPtr(false),
				},
				{
					Function: "b",
					Package:  "test.package",
					InApp: testutil.BoolPtr(false),
				},
				{
					Function: "c",
					Package:  "test.package",
					InApp: testutil.BoolPtr(true),
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
		Transaction: transaction.Transaction{
			ActiveThreadID: 0,
		},
	},
} // end prof definition

var secondSampledProfile = sample.Profile{
	RawProfile: sample.RawProfile{
		EventID:  "cd2",
		Platform: platform.Cocoa,
		Version:  "1",
		Trace: sample.Trace{
			Frames: []frame.Frame{
				{
					Function: "a",
					Package:  "test.package",
					InApp: testutil.BoolPtr(false),
				},
				{
					Function: "c",
					Package:  "test.package",
					InApp: testutil.BoolPtr(true),
				},
				{
					Function: "e",
					Package:  "test.package",
					InApp: testutil.BoolPtr(false),
				},
				{
					Function: "b",
					Package:  "test.package",
					InApp: testutil.BoolPtr(false),
				},
			}, //end frames
			Stacks: []sample.Stack{
				{0, 1}, // a,c
				{2},    // e
				{3, 0}, // b,a
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
			}, // end Samples
		}, // end Trace
		Transaction: transaction.Transaction{
			ActiveThreadID: 0,
		},
	},
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
	addCallTreeToFlamegraph(&flamegraphTree, callTrees[0], pr.ID())

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

	addCallTreeToFlamegraph(&flamegraphTree, callTrees[0], pr.ID())

	sp := toSpeedscope(flamegraphTree, 1)
	prof := sp.Profiles[0].(speedscope.SampledProfile)

	expectedSamples := [][]int{
		{0, 1}, // [a, b]   prof_id[0, 1]
		{0},    // [a]      prof_id[0]
		{2, 0}, // [c, a]   prof_id[1]
		{2},    // [c]      prof_id[0]
		{3},    // [e]      prof_id[1]
	}

	expectedSamplesProfiles := []map[string]struct{}{
		{firstSampledProfile.EventID: void, secondSampledProfile.EventID: void},
		{firstSampledProfile.EventID: void},
		{secondSampledProfile.EventID: void},
		{firstSampledProfile.EventID: void},
		{secondSampledProfile.EventID: void},
	}

	expectedWeights := []uint64{3, 1, 1, 1, 1}

	if diff := testutil.Diff(expectedSamples, prof.Samples); diff != "" {
		t.Fatalf("expected \"%v\" found \"%v\"", expectedSamples, prof.Samples)
	}

	if diff := testutil.Diff(expectedWeights, prof.Weights); diff != "" {
		t.Fatalf("expected \"%v\" found \"%v\"", expectedWeights, prof.Weights)
	}

	actualSamplesProfiles := getProfilesIDsfromIndexes(prof.SamplesProfiles, sp.Shared.ProfileIDs)
	if diff := testutil.Diff(expectedSamplesProfiles, actualSamplesProfiles); diff != "" {
		t.Fatalf("expected \"%v\" found \"%v\"", expectedSamplesProfiles, actualSamplesProfiles)
	}

	appFrames := getApplicationFrames(sp.Shared.Frames)
	if len(appFrames) != 1 {
		t.Fatalf("expected 1 application frame, found %d", len(appFrames))
	}


	if len(appFrames) > 0 && appFrames[0].Name != "c" {
		t.Fatalf("expected frame name \"c\", found \"%s\"", appFrames[0].Name)
	}
}

func getProfilesIDsfromIndexes(sampleProfilesIDX [][]int, profileIDs []string) []map[string]struct{} {
	samplesProfilesIDs := make([]map[string]struct{}, 0, len(sampleProfilesIDX))
	for _, sample := range sampleProfilesIDX {
		IDs := make(map[string]struct{})
		for _, idx := range sample {
			IDs[profileIDs[idx]] = void
		}
		samplesProfilesIDs = append(samplesProfilesIDs, IDs)
	}
	return samplesProfilesIDs
}

func getApplicationFrames(frames []speedscope.Frame) ([]speedscope.Frame) {
	_frames := []speedscope.Frame{}
	for _, v:= range frames {
		if v.IsApplication {
			_frames = append(_frames, v)
		}
	}

	return _frames
}