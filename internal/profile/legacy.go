package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/speedscope"
)

type (
	LegacyProfile struct {
		RawProfile

		Trace Trace `json:"profile"`
	}

	RawProfile struct {
		AndroidAPILevel      uint32          `json:"android_api_level,omitempty"`
		Architecture         string          `json:"architecture,omitempty"`
		DebugMeta            interface{}     `json:"debug_meta,omitempty"`
		DeviceClassification string          `json:"device_classification"`
		DeviceLocale         string          `json:"device_locale"`
		DeviceManufacturer   string          `json:"device_manufacturer"`
		DeviceModel          string          `json:"device_model"`
		DeviceOSBuildNumber  string          `json:"device_os_build_number,omitempty"`
		DeviceOSName         string          `json:"device_os_name"`
		DeviceOSVersion      string          `json:"device_os_version"`
		DurationNS           uint64          `json:"duration_ns"`
		Environment          string          `json:"environment,omitempty"`
		OrganizationID       uint64          `json:"organization_id"`
		Platform             string          `json:"platform"`
		Profile              json.RawMessage `json:"profile,omitempty"`
		ProfileID            string          `json:"profile_id"`
		ProjectID            uint64          `json:"project_id"`
		Received             time.Time       `json:"received"`
		TraceID              string          `json:"trace_id"`
		TransactionID        string          `json:"transaction_id"`
		TransactionName      string          `json:"transaction_name"`
		VersionCode          string          `json:"version_code"`
		VersionName          string          `json:"version_name"`
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
	case "android":
		var t Android
		err := json.Unmarshal(raw, &t)
		if err != nil {
			return err
		}
		(*p).Trace = t
	case "python":
		var t Python
		err := json.Unmarshal(raw, &t)
		if err != nil {
			return err
		}
		(*p).Trace = t
	case "rust":
		var t Rust
		err := json.Unmarshal(raw, &t)
		if err != nil {
			return err
		}
		(*p).Trace = t
	case "node", "typescript":
		return nil
	default:
		return errors.New("unknown platform")
	}
	p.Profile = nil
	return nil
}

func (p LegacyProfile) CallTrees() (map[uint64][]*nodetree.Node, error) {
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
