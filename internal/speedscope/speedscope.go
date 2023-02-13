package speedscope

import (
	"encoding/json"

	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/measurements"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/timeutil"
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
		EndValue     uint64           `json:"endValue,omitempty"`
		IsMainThread bool             `json:"isMainThread"`
		Name         string           `json:"name"`
		Priority     int              `json:"priority,omitempty"`
		Queues       map[string]Queue `json:"queues,omitempty"`
		Samples      [][]int          `json:"samples"`
		StartValue   uint64           `json:"startValue,omitempty"`
		State        string           `json:"state,omitempty"`
		ThreadID     uint64           `json:"threadID,omitempty"`
		Type         ProfileType      `json:"type"`
		Unit         ValueUnit        `json:"unit"`
		Weights      []uint64         `json:"weights"`
	}

	SharedData struct {
		Frames []Frame `json:"frames"`
	}

	EventType   string
	ProfileType string
	ValueUnit   string

	Output struct {
		ActiveProfileIndex int                                 `json:"activeProfileIndex,omitempty"`
		AndroidClock       string                              `json:"androidClock,omitempty"`
		DurationNS         uint64                              `json:"durationNS,omitempty"`
		Images             []debugmeta.Image                   `json:"images,omitempty"`
		Measurements       map[string]measurements.Measurement `json:"measurements,omitempty"`
		Metadata           ProfileMetadata                     `json:"metadata,omitempty"`
		Platform           platform.Platform                   `json:"platform"`
		ProfileID          string                              `json:"profileID,omitempty"`
		Profiles           []interface{}                       `json:"profiles"`
		ProjectID          uint64                              `json:"projectID"`
		Shared             SharedData                          `json:"shared"`
		TransactionName    string                              `json:"transactionName"`
		Version            string                              `json:"version,omitempty"`
	}

	ProfileMetadata struct {
		ProfileView

		Timestamp timeutil.Time `json:"timestamp,omitempty"`
		Version   string        `json:"version"`
	}

	ProfileView struct {
		AndroidAPILevel      uint32                              `json:"androidAPILevel,omitempty"`
		Architecture         string                              `json:"architecture,omitempty"`
		DebugMeta            interface{}                         `json:"-"`
		DeviceClassification string                              `json:"deviceClassification"`
		DeviceLocale         string                              `json:"deviceLocale"`
		DeviceManufacturer   string                              `json:"deviceManufacturer"`
		DeviceModel          string                              `json:"deviceModel"`
		DeviceOSBuildNumber  string                              `json:"deviceOSBuildNumber,omitempty"`
		DeviceOSName         string                              `json:"deviceOSName"`
		DeviceOSVersion      string                              `json:"deviceOSVersion"`
		DurationNS           uint64                              `json:"durationNS"`
		Environment          string                              `json:"environment,omitempty"`
		Measurements         map[string]measurements.Measurement `json:"-"`
		OrganizationID       uint64                              `json:"organizationID"`
		Platform             platform.Platform                   `json:"platform"`
		Profile              json.RawMessage                     `json:"-"`
		ProfileID            string                              `json:"profileID"`
		ProjectID            uint64                              `json:"projectID"`
		Received             timeutil.Time                       `json:"received"`
		RetentionDays        int                                 `json:"-"`
		TraceID              string                              `json:"traceID"`
		TransactionID        string                              `json:"transactionID"`
		TransactionName      string                              `json:"transactionName"`
		VersionCode          string                              `json:"-"`
		VersionName          string                              `json:"-"`
	}
)
