package flamegraph

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/testutil"
	"github.com/getsentry/vroom/internal/timeutil"
	"github.com/getsentry/vroom/internal/transaction"
	"github.com/google/go-cmp/cmp"
)

var falseValue = false
var trueValue = true

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
					InApp:    &falseValue,
				},
				{
					Function: "b",
					Package:  "test.package",
					InApp:    &falseValue,
				},
				{
					Function: "c",
					Package:  "test.package",
					InApp:    &trueValue,
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
					InApp:    &falseValue,
				},
				{
					Function: "c",
					Package:  "test.package",
					InApp:    &trueValue,
				},
				{
					Function: "e",
					Package:  "test.package",
					InApp:    &falseValue,
				},
				{
					Function: "b",
					Package:  "test.package",
					InApp:    &falseValue,
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

var expectedSp = speedscope.Output{
	ActiveProfileIndex: 0,
	AndroidClock:       "",
	DurationNS:         0,
	Images:             nil,
	Measurements:       nil,
	Metadata: speedscope.ProfileMetadata{
		ProfileView: speedscope.ProfileView{
			AndroidAPILevel:      0,
			Architecture:         "",
			DebugMeta:            debugmeta.DebugMeta{},
			DeviceClassification: "",
			DeviceLocale:         "",
			DeviceManufacturer:   "",
			DeviceModel:          "",
			DeviceOSBuildNumber:  "",
			DeviceOSName:         "",
			DeviceOSVersion:      "",
			DurationNS:           0,
			Environment:          "",
			Measurements:         nil,
			OrganizationID:       0,
			Platform:             "",
			Profile:              nil,
			ProfileID:            "",
			ProjectID:            0,
			Received:             timeutil.Time(time.Time{}),
			RetentionDays:        0,
			TraceID:              "",
			TransactionID:        "",
			VersionCode:          "",
			VersionName:          "",
		},
		Timestamp: timeutil.Time(time.Time{}),
		Version:   "",
	},
	Platform:  "",
	ProfileID: "",
	Profiles: []interface{}{
		speedscope.SampledProfile{
			EndValue:     7,
			IsMainThread: true,
			Name:         "",
			Priority:     0,
			Queues:       nil,
			Samples: [][]int{
				{0, 1},
				{0},
				{2, 0},
				{2},
				{3},
			},
			SamplesProfiles: [][]int{
				{0, 1},
				{0},
				{1},
				{0},
				{1},
			},
			StartValue: 0,
			State:      "",
			ThreadID:   0,
			Type:       "sampled",
			Unit:       "count",
			Weights:    []uint64{3, 1, 1, 1, 1}},
	},
	ProjectID: 0,
	Shared: speedscope.SharedData{
		Frames: []speedscope.Frame{
			{Image: "test.package", Name: "a"}, {Image: "test.package", Name: "b"},
			{Image: "test.package", IsApplication: true, Name: "c"},
			{Image: "test.package", Name: "e"},
		},
		ProfileIDs: []string{"ab1", "cd2"},
	},
	TransactionName: "",
	Version:         "",
}

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

	if diff := testutil.Diff(expectedSp, sp, cmp.AllowUnexported(timeutil.Time{})); diff != "" {
		t.Fatalf("expected \"%+v\" found \"%+v\"", expectedSp, sp)
	}
}
