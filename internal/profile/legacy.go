package profile

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/measurements"
	"github.com/getsentry/vroom/internal/metadata"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/timeutil"
	"github.com/getsentry/vroom/internal/transaction"
)

type (
	LegacyProfile struct {
		AndroidAPILevel      uint32                              `json:"android_api_level,omitempty"`
		Architecture         string                              `json:"architecture,omitempty"`
		BuildID              string                              `json:"build_id,omitempty"`
		DebugMeta            debugmeta.DebugMeta                 `json:"debug_meta,omitempty"`
		DeviceClassification string                              `json:"device_classification"`
		DeviceLocale         string                              `json:"device_locale"`
		DeviceManufacturer   string                              `json:"device_manufacturer"`
		DeviceModel          string                              `json:"device_model"`
		DeviceOSBuildNumber  string                              `json:"device_os_build_number,omitempty"`
		DeviceOSName         string                              `json:"device_os_name"`
		DeviceOSVersion      string                              `json:"device_os_version"`
		DurationNS           uint64                              `json:"duration_ns"`
		Environment          string                              `json:"environment,omitempty"`
		Measurements         map[string]measurements.Measurement `json:"measurements,omitempty"`
		OrganizationID       uint64                              `json:"organization_id"`
		Platform             platform.Platform                   `json:"platform"`
		Profile              json.RawMessage                     `json:"profile,omitempty"`
		ProfileID            string                              `json:"profile_id"`
		ProjectID            uint64                              `json:"project_id"`
		Received             timeutil.Time                       `json:"received"`
		RetentionDays        int                                 `json:"retention_days"`
		TraceID              string                              `json:"trace_id"`
		TransactionID        string                              `json:"transaction_id"`
		TransactionMetadata  transaction.Metadata                `json:"transaction_metadata"`
		TransactionName      string                              `json:"transaction_name"`
		TransactionTags      map[string]string                   `json:"transaction_tags,omitempty"`
		VersionCode          string                              `json:"version_code"`
		VersionName          string                              `json:"version_name"`
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
	return fmt.Sprintf(
		"%d/%d/%s",
		organizationID,
		projectID,
		strings.ReplaceAll(profileID, "-", ""),
	)
}

func (p LegacyProfile) StoragePath() string {
	return StoragePath(p.OrganizationID, p.ProjectID, p.ProfileID)
}

func (p LegacyProfile) IsSampleFormat() bool {
	return false
}

func (p *LegacyProfile) Metadata() metadata.Metadata {
	return metadata.Metadata{
		AndroidAPILevel:      p.AndroidAPILevel,
		Architecture:         "unknown",
		DeviceClassification: p.DeviceClassification,
		DeviceLocale:         p.DeviceLocale,
		DeviceManufacturer:   p.DeviceManufacturer,
		DeviceModel:          p.DeviceModel,
		DeviceOSBuildNumber:  p.DeviceOSBuildNumber,
		DeviceOSName:         p.DeviceOSName,
		DeviceOSVersion:      p.DeviceOSVersion,
		ID:                   p.ProfileID,
		ProjectID:            strconv.FormatUint(p.GetProjectID(), 10),
		Timestamp:            p.Received.Time().Unix(),
		TraceDurationMs:      float64(p.DurationNS) / 1_000_000,
		TransactionID:        p.TransactionID,
		TransactionName:      p.TransactionName,
		VersionCode:          p.VersionCode,
		VersionName:          p.VersionName,
	}
}

func (p LegacyProfile) GetPlatform() platform.Platform {
	return p.Platform
}

func (p LegacyProfile) GetEnvironment() string {
	return p.Environment
}

func (p LegacyProfile) GetDebugMeta() debugmeta.DebugMeta {
	return p.DebugMeta
}

func (p LegacyProfile) GetTimestamp() time.Time {
	return p.Received.Time()
}

func (p LegacyProfile) GetReceived() time.Time {
	return p.Received.Time()
}

func (p LegacyProfile) GetRelease() string {
	return FormatVersion(p.VersionName, p.VersionCode)
}

func (p LegacyProfile) GetRetentionDays() int {
	return p.RetentionDays
}

func (p LegacyProfile) GetTransactionMetadata() transaction.Metadata {
	return p.TransactionMetadata
}

func (p LegacyProfile) GetTransactionTags() map[string]string {
	return p.TransactionTags
}
