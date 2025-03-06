package speedscope

import (
	"encoding/json"
	"time"

	"github.com/getsentry/vroom/internal/clientsdk"
	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/measurements"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/timeutil"
	"github.com/getsentry/vroom/internal/transaction"
	"github.com/getsentry/vroom/internal/utils"
)

const (
	ValueUnitNanoseconds ValueUnit = "nanoseconds"
	ValueUnitCount       ValueUnit = "count"

	EventTypeOpenFrame  EventType = "O"
	EventTypeCloseFrame EventType = "C"

	ProfileTypeEvented ProfileType = "evented"
	ProfileTypeSampled ProfileType = "sampled"
)

type (
	Frame struct {
		Col           uint32 `json:"col,omitempty"`
		File          string `json:"file,omitempty"`
		Image         string `json:"image,omitempty"`
		Inline        bool   `json:"inline,omitempty"`
		IsApplication bool   `json:"is_application"`
		Line          uint32 `json:"line,omitempty"`
		Name          string `json:"name"`
		Path          string `json:"path,omitempty"`
	}

	Event struct {
		Type  EventType `json:"type"`
		Frame int       `json:"frame"`
		At    uint64    `json:"at"`
	}

	Queue struct {
		Label   string `json:"name"`
		StartNS uint64 `json:"start_ns"`
		EndNS   uint64 `json:"end_ns"`
	}

	EventedProfile struct {
		EndValue   uint64      `json:"endValue"`
		Events     []Event     `json:"events"`
		Name       string      `json:"name"`
		StartValue uint64      `json:"startValue"`
		ThreadID   uint64      `json:"threadID"`
		Type       ProfileType `json:"type"`
		Unit       ValueUnit   `json:"unit"`
	}

	SampledProfile struct {
		EndValue          uint64           `json:"endValue"`
		IsMainThread      bool             `json:"isMainThread"`
		Name              string           `json:"name"`
		Priority          int              `json:"priority,omitempty"`
		Queues            map[string]Queue `json:"queues,omitempty"`
		Samples           [][]int          `json:"samples"`
		SamplesProfiles   [][]int          `json:"samples_profiles,omitempty"`
		SamplesExamples   [][]int          `json:"samples_examples,omitempty"`
		StartValue        uint64           `json:"startValue"`
		State             string           `json:"state,omitempty"`
		ThreadID          uint64           `json:"threadID"`
		Type              ProfileType      `json:"type"`
		Unit              ValueUnit        `json:"unit"`
		Weights           []uint64         `json:"weights"`
		SampleDurationsNs []uint64         `json:"sample_durations_ns"`
		SampleCounts      []uint64         `json:"sample_counts,omitempty"`
	}

	SharedData struct {
		Frames     []Frame                 `json:"frames"`
		ProfileIDs []string                `json:"profile_ids,omitempty"`
		Profiles   []utils.ExampleMetadata `json:"profiles,omitempty"`
	}

	EventType   string
	ProfileType string
	ValueUnit   string

	Output struct {
		ActiveProfileIndex int                      `json:"activeProfileIndex"`
		AndroidClock       string                   `json:"androidClock,omitempty"`
		DurationNS         uint64                   `json:"durationNS,omitempty"`
		Images             []debugmeta.Image        `json:"images,omitempty"`
		Measurements       interface{}              `json:"measurements,omitempty"`
		Metadata           ProfileMetadata          `json:"metadata"`
		Platform           platform.Platform        `json:"platform"`
		ProfileID          string                   `json:"profileID,omitempty"`
		ChunkID            string                   `json:"chunkID,omitempty"`
		Profiles           []interface{}            `json:"profiles"`
		ProjectID          uint64                   `json:"projectID"`
		Shared             SharedData               `json:"shared"`
		TransactionName    string                   `json:"transactionName"`
		Version            string                   `json:"version,omitempty"`
		Metrics            *[]utils.FunctionMetrics `json:"metrics"`
	}

	ProfileMetadata struct {
		ProfileView

		Version string `json:"version"`
	}

	ProfileView struct {
		AndroidAPILevel      uint32                              `json:"androidAPILevel,omitempty"` //nolint:unused
		Architecture         string                              `json:"architecture,omitempty"`    //nolint:unused
		BuildID              string                              `json:"-"`                         //nolint:unused
		ClientSDK            clientsdk.ClientSDK                 `json:"-"`
		DebugMeta            debugmeta.DebugMeta                 `json:"-"`                             //nolint:unused
		DeviceClassification string                              `json:"deviceClassification"`          //nolint:unused
		DeviceLocale         string                              `json:"deviceLocale"`                  //nolint:unused
		DeviceManufacturer   string                              `json:"deviceManufacturer"`            //nolint:unused
		DeviceModel          string                              `json:"deviceModel"`                   //nolint:unused
		DeviceOSBuildNumber  string                              `json:"deviceOSBuildNumber,omitempty"` //nolint:unused
		DeviceOSName         string                              `json:"deviceOSName"`                  //nolint:unused
		DeviceOSVersion      string                              `json:"deviceOSVersion"`               //nolint:unused
		DurationNS           uint64                              `json:"durationNS"`                    //nolint:unused
		Environment          string                              `json:"environment,omitempty"`         //nolint:unused
		JsProfile            json.RawMessage                     `json:"-"`                             //nolint:unused
		Measurements         map[string]measurements.Measurement `json:"-"`                             //nolint:unused
		Options              utils.Options                       `json:"-"`                             //nolint:unused
		OrganizationID       uint64                              `json:"organizationID"`
		Platform             platform.Platform                   `json:"platform"`            //nolint:unused
		Profile              json.RawMessage                     `json:"-"`                   //nolint:unused
		ProfileID            string                              `json:"profileID"`           //nolint:unused
		ProjectID            uint64                              `json:"projectID"`           //nolint:unused
		Received             timeutil.Time                       `json:"received"`            //nolint:unused
		RetentionDays        int                                 `json:"-"`                   //nolint:unused
		Sampled              bool                                `json:"sampled"`             //nolint:unused
		Timestamp            time.Time                           `json:"timestamp,omitempty"` //nolint:unused
		TraceID              string                              `json:"traceID"`             //nolint:unused
		TransactionID        string                              `json:"transactionID"`       //nolint:unused
		TransactionMetadata  transaction.Metadata                `json:"-"`                   //nolint:unused
		TransactionName      string                              `json:"transactionName"`     //nolint:unused
		TransactionTags      map[string]string                   `json:"-"`                   //nolint:unused
		VersionCode          string                              `json:"-"`                   //nolint:unused
		VersionName          string                              `json:"-"`                   //nolint:unused
	}
)
