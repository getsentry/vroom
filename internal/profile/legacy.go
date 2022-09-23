package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/nodetree"
)

type (
	LegacyProfile struct {
		AndroidAPILevel      uint32      `json:"android_api_level,omitempty"`
		Architecture         string      `json:"architecture,omitempty"`
		DebugMeta            interface{} `json:"debug_meta,omitempty"`
		DeviceClassification string      `json:"device_classification"`
		DeviceLocale         string      `json:"device_locale"`
		DeviceManufacturer   string      `json:"device_manufacturer"`
		DeviceModel          string      `json:"device_model"`
		DeviceOSBuildNumber  string      `json:"device_os_build_number,omitempty"`
		DeviceOSName         string      `json:"device_os_name"`
		DeviceOSVersion      string      `json:"device_os_version"`
		DurationNS           uint64      `json:"duration_ns"`
		Environment          string      `json:"environment,omitempty"`
		OrganizationID       uint64      `json:"organization_id"`
		Platform             string      `json:"platform"`
		Profile              string      `json:"profile,omitempty"`
		ProfileID            string      `json:"profile_id"`
		ProjectID            uint64      `json:"project_id"`
		Received             time.Time   `json:"received"`
		Trace                Trace       `json:"-"`
		TraceID              string      `json:"trace_id"`
		TransactionID        string      `json:"transaction_id"`
		TransactionName      string      `json:"transaction_name"`
		VersionCode          string      `json:"version_code"`
		VersionName          string      `json:"version_name"`
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

func (p LegacyProfile) CallTrees() (map[uint64][]*nodetree.Node, error) {
	var a Trace
	switch p.Platform {
	case "cocoa":
		var cp IOS
		err := json.Unmarshal([]byte(p.Profile), &cp)
		if err != nil {
			return nil, err
		}
		cp.ReplaceIdleStacks()
		a = cp
	case "android":
		var ap Android
		err := json.Unmarshal([]byte(p.Profile), &ap)
		if err != nil {
			return nil, err
		}
		a = ap
	case "python", "rust", "node", "typescript":
		return nil, nil
	default:
		return nil, errors.New("unknown platform")
	}
	return a.CallTrees(), nil
}
