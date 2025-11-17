package main

import (
	"context"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/segmentio/kafka-go"
)

type (
	// FunctionsKafkaMessage is representing the struct we send to Kafka to insert functions in ClickHouse.
	FunctionsKafkaMessage struct {
		Environment            string                      `json:"environment,omitempty"`
		Functions              []nodetree.CallTreeFunction `json:"functions"`
		ID                     string                      `json:"profile_id"`
		Platform               platform.Platform           `json:"platform"`
		ProjectID              uint64                      `json:"project_id"`
		Received               int64                       `json:"received"`
		Release                string                      `json:"release,omitempty"`
		RetentionDays          int                         `json:"retention_days"`
		Timestamp              int64                       `json:"timestamp"`
		TransactionName        string                      `json:"transaction_name"`
		StartTimestamp         float64                     `json:"start_timestamp,omitempty"`
		EndTimestamp           float64                     `json:"end_timestamp,omitempty"`
		ProfilingType          string                      `json:"profiling_type,omitempty"`
		MaterializationVersion uint8                       `json:"materialization_version"`
	}

	// ProfileKafkaMessage is representing the struct we send to Kafka to insert a profile in ClickHouse.
	ProfileKafkaMessage struct {
		AndroidAPILevel      uint32            `json:"android_api_level,omitempty"`
		Architecture         string            `json:"architecture,omitempty"`
		DeviceClassification string            `json:"device_classification,omitempty"`
		DeviceLocale         string            `json:"device_locale"`
		DeviceManufacturer   string            `json:"device_manufacturer"`
		DeviceModel          string            `json:"device_model"`
		DeviceOSBuildNumber  string            `json:"device_os_build_number,omitempty"`
		DeviceOSName         string            `json:"device_os_name"`
		DeviceOSVersion      string            `json:"device_os_version"`
		DurationNS           uint64            `json:"duration_ns"`
		Environment          string            `json:"environment,omitempty"`
		ID                   string            `json:"profile_id"`
		OrganizationID       uint64            `json:"organization_id"`
		Platform             platform.Platform `json:"platform"`
		ProjectID            uint64            `json:"project_id"`
		Received             int64             `json:"received"`
		RetentionDays        int               `json:"retention_days"`
		SDKName              string            `json:"sdk_name,omitempty"`
		SDKVersion           string            `json:"sdk_version,omitempty"`
		TraceID              string            `json:"trace_id"`
		TransactionID        string            `json:"transaction_id"`
		TransactionName      string            `json:"transaction_name"`
		VersionCode          string            `json:"version_code"`
		VersionName          string            `json:"version_name"`
	}
)
type KafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}
