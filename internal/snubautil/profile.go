package snubautil

import (
	"time"
)

type Profile struct {
	AndroidAPILevel      uint32    `json:"android_api_level,omitempty"`
	DeviceClassification string    `json:"device_classification"`
	DeviceLocale         string    `json:"device_locale"`
	DeviceManufacturer   string    `json:"device_manufacturer"`
	DeviceModel          string    `json:"device_model"`
	DeviceOsBuildNumber  string    `json:"device_os_build_number,omitempty"`
	DeviceOsName         string    `json:"device_os_name"`
	DeviceOsVersion      string    `json:"device_os_version"`
	DurationNs           uint64    `json:"duration_ns"`
	Environment          string    `json:"environment,omitempty"`
	OrganizationID       uint64    `json:"organization_id"`
	Platform             string    `json:"platform"`
	Profile              string    `json:"-"`
	ProfileID            string    `json:"profile_id"`
	ProjectID            uint64    `json:"project_id"`
	Received             time.Time `json:"received"`
	TraceID              string    `json:"trace_id"`
	TransactionID        string    `json:"transaction_id"`
	TransactionName      string    `json:"transaction_name"`
	VersionCode          string    `json:"version_code"`
	VersionName          string    `json:"version_name"`
}

func (p Profile) Version() string {
	return FormatVersion(p.VersionName, p.VersionCode)
}
