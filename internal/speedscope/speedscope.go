package speedscope

import (
	"encoding/json"
	"time"

	"github.com/getsentry/vroom/internal/debugmeta"
)

const (
	ValueUnitNanoseconds ValueUnit = "nanoseconds"

	EventTypeOpenFrame  EventType = "O"
	EventTypeCloseFrame EventType = "C"

	ProfileTypeEvented ProfileType = "evented"
	ProfileTypeSampled ProfileType = "sampled"
)

type (
	Frame struct {
		Col           int    `json:"col,omitempty"`
		File          string `json:"file,omitempty"`
		Image         string `json:"image,omitempty"`
		Inline        bool   `json:"inline,omitempty"`
		IsApplication bool   `json:"is_application"`
		Line          uint32 `json:"line,omitempty"`
		Name          string `json:"name"`
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
		EndValue     uint64            `json:"endValue"`
		IsMainThread bool              `json:"isMainThread"`
		Images       []debugmeta.Image `json:"images,omitempty"`
		Name         string            `json:"name"`
		Priority     int               `json:"priority"`
		Queues       map[string]Queue  `json:"queues,omitempty"`
		Samples      [][]int           `json:"samples"`
		StartValue   uint64            `json:"startValue"`
		State        string            `json:"state,omitempty"`
		ThreadID     uint64            `json:"threadID"`
		Type         ProfileType       `json:"type"`
		Unit         ValueUnit         `json:"unit"`
		Weights      []uint64          `json:"weights"`
	}

	SharedData struct {
		Frames []Frame `json:"frames"`
	}

	EventType   string
	ProfileType string
	ValueUnit   string

	Output struct {
		ActiveProfileIndex int             `json:"activeProfileIndex"`
		AndroidClock       string          `json:"androidClock,omitempty"`
		DurationNS         uint64          `json:"durationNS"`
		Metadata           ProfileMetadata `json:"metadata"`
		Platform           string          `json:"platform"`
		ProfileID          string          `json:"profileID"`
		Profiles           []interface{}   `json:"profiles"`
		ProjectID          uint64          `json:"projectID"`
		Shared             SharedData      `json:"shared"`
		TransactionName    string          `json:"transactionName"`
		Version            string          `json:"version"`
	}

	ProfileMetadata struct {
		ProfileView

		Version string `json:"version"`
	}

	ProfileView struct {
		AndroidAPILevel      uint32          `json:"androidAPILevel,omitempty"`
		Architecture         string          `json:"architecture,omitempty"`
		DebugMeta            interface{}     `json:"-"`
		DeviceClassification string          `json:"deviceClassification"`
		DeviceLocale         string          `json:"deviceLocale"`
		DeviceManufacturer   string          `json:"deviceManufacturer"`
		DeviceModel          string          `json:"deviceModel"`
		DeviceOSBuildNumber  string          `json:"deviceOSBuildNumber,omitempty"`
		DeviceOSName         string          `json:"deviceOSName"`
		DeviceOSVersion      string          `json:"deviceOSVersion"`
		DurationNS           uint64          `json:"durationNS"`
		Environment          string          `json:"environment,omitempty"`
		OrganizationID       uint64          `json:"organizationID"`
		Platform             string          `json:"platform"`
		Profile              json.RawMessage `json:"-"`
		ProfileID            string          `json:"profileID"`
		ProjectID            uint64          `json:"projectID"`
		Received             time.Time       `json:"received"`
		TraceID              string          `json:"traceID"`
		TransactionID        string          `json:"transactionID"`
		TransactionName      string          `json:"transactionName"`
		VersionCode          string          `json:"-"`
		VersionName          string          `json:"-"`
	}
)
