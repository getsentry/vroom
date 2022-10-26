package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/measurements"
	"github.com/getsentry/vroom/internal/metadata"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/speedscope"
)

var (
	ErrProfileHasNoTrace = errors.New("profile has no trace")
)

type (
	LegacyProfile struct {
		RawProfile

		Trace Trace `json:"profile"`
	}

	RawProfile struct {
		AndroidAPILevel      uint32                                `json:"android_api_level,omitempty"`
		Architecture         string                                `json:"architecture,omitempty"`
		DebugMeta            interface{}                           `json:"debug_meta,omitempty"`
		DeviceClassification string                                `json:"device_classification"`
		DeviceLocale         string                                `json:"device_locale"`
		DeviceManufacturer   string                                `json:"device_manufacturer"`
		DeviceModel          string                                `json:"device_model"`
		DeviceOSBuildNumber  string                                `json:"device_os_build_number,omitempty"`
		DeviceOSName         string                                `json:"device_os_name"`
		DeviceOSVersion      string                                `json:"device_os_version"`
		DurationNS           uint64                                `json:"duration_ns"`
		Environment          string                                `json:"environment,omitempty"`
		OrganizationID       uint64                                `json:"organization_id"`
		Platform             string                                `json:"platform"`
		Profile              json.RawMessage                       `json:"profile,omitempty"`
		ProfileID            string                                `json:"profile_id"`
		ProjectID            uint64                                `json:"project_id"`
		Received             time.Time                             `json:"received"`
		TraceID              string                                `json:"trace_id"`
		TransactionID        string                                `json:"transaction_id"`
		TransactionName      string                                `json:"transaction_name"`
		VersionCode          string                                `json:"version_code"`
		VersionName          string                                `json:"version_name"`
		Measurements         map[string][]measurements.Measurement `json:"measurements,omitempty"`
	}
)

func (p LegacyProfile) GetOrganizationID() uint64 {
	return p.OrganizationID
}

func (p LegacyProfile) GetProjectID() uint64 {
	return p.ProjectID
}

func (p LegacyProfile) GetID() string {
	return p.ProfileID
}

func (p LegacyProfile) Version() string {
	return FormatVersion(p.VersionName, p.VersionCode)
}

func StoragePath(organizationID, projectID uint64, profileID string) string {
	return fmt.Sprintf("%d/%d/%s", organizationID, projectID, strings.Replace(profileID, "-", "", -1))
}

func (p LegacyProfile) StoragePath() string {
	return StoragePath(p.OrganizationID, p.ProjectID, p.ProfileID)
}

func (p *LegacyProfile) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &p.RawProfile)
	if err != nil {
		return err
	}
	// when reading a profile from Snuba, there's no profile attached
	if len(p.Profile) == 0 {
		return nil
	}
	var raw []byte
	if p.Profile[0] == '"' {
		var s string
		err := json.Unmarshal(p.Profile, &s)
		if err != nil {
			return err
		}
		raw = []byte(s)
	} else {
		raw = p.Profile
	}
	switch p.Platform {
	case "cocoa":
		var t IOS
		err := json.Unmarshal(raw, &t)
		if err != nil {
			return err
		}
		t.ReplaceIdleStacks()
		(*p).Trace = t
		p.Profile = nil
	case "android":
		var t Android
		err := json.Unmarshal(raw, &t)
		if err != nil {
			return err
		}
		(*p).Trace = t
		p.Profile = nil
	case "python":
		var t Python
		err := json.Unmarshal(raw, &t)
		if err != nil {
			return err
		}
		(*p).Trace = t
		p.Profile = nil
	case "rust":
		var t Rust
		err := json.Unmarshal(raw, &t)
		if err != nil {
			return err
		}
		(*p).Trace = t
		p.Profile = nil
	case "typescript":
		(*p).Trace = new(Typescript)
	default:
		return errors.New("unknown platform")
	}
	return nil
}

func (p LegacyProfile) CallTrees() (map[uint64][]*nodetree.Node, error) {
	// Profiles longer than 5s contain a lot of call trees and it produces a lot of noise for the aggregation.
	// The majority of them might also be timing out and we want to ignore them for the aggregation.
	if time.Duration(p.DurationNS) > 5*time.Second {
		return make(map[uint64][]*nodetree.Node), nil
	}
	if p.Trace == nil {
		return nil, ErrProfileHasNoTrace
	}
	return p.Trace.CallTrees(), nil
}

func (p *LegacyProfile) Speedscope() (speedscope.Output, error) {
	o, err := p.Trace.Speedscope()
	if err != nil {
		return speedscope.Output{}, err
	}

	version := FormatVersion(p.VersionName, p.VersionCode)

	o.DurationNS = p.DurationNS
	o.Metadata = speedscope.ProfileMetadata{ProfileView: speedscope.ProfileView(p.RawProfile), Version: version}
	o.Platform = p.Platform
	o.ProfileID = p.ProfileID
	o.ProjectID = p.ProjectID
	o.TransactionName = p.TransactionName
	o.Version = version

	return o, nil
}

func (p *LegacyProfile) Metadata() metadata.Metadata {
	return metadata.Metadata{
		AndroidAPILevel:      p.AndroidAPILevel,
		DeviceClassification: p.DeviceClassification,
		DeviceLocale:         p.DeviceLocale,
		DeviceManufacturer:   p.DeviceManufacturer,
		DeviceModel:          p.DeviceModel,
		DeviceOsBuildNumber:  p.DeviceOSBuildNumber,
		DeviceOsName:         p.DeviceOSName,
		DeviceOsVersion:      p.DeviceOSVersion,
		ID:                   p.ProfileID,
		ProjectID:            strconv.FormatUint(p.GetProjectID(), 10),
		Timestamp:            p.Received.Unix(),
		TraceDurationMs:      float64(p.DurationNS) / 1_000_000,
		TransactionID:        p.TransactionID,
		TransactionName:      p.TransactionName,
		VersionCode:          p.VersionCode,
		VersionName:          p.VersionName,
	}
}

func (p *LegacyProfile) GetPlatform() string {
	return p.Platform
}

func (p *LegacyProfile) Raw() []byte {
	return p.Profile
}
