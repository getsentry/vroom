package main

import (
	"context"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/segmentio/kafka-go"
)

type (
	// FunctionsKafkaMessage is representing the struct we send to Kafka to insert functions in ClickHouse.
	FunctionsKafkaMessage struct {
		Environment     string                      `json:"environment,omitempty"`
		Functions       []nodetree.CallTreeFunction `json:"functions"`
		ID              string                      `json:"profile_id"`
		Platform        platform.Platform           `json:"platform"`
		ProjectID       uint64                      `json:"project_id"`
		Received        int64                       `json:"received"`
		Release         string                      `json:"release,omitempty"`
		RetentionDays   int                         `json:"retention_days"`
		Timestamp       int64                       `json:"timestamp"`
		TransactionName string                      `json:"transaction_name"`
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

	// MetricsSummaryKafkaMessage is representing the struct we send to Kafka to insert Metrics Summary in ClickHouse.
	MetricsSummaryKafkaMessage struct {
		Count         uint64            `json:"count"`
		DurationMs    uint32            `json:"duration_ms"`
		EndTimestamp  float64           `json:"end_timestamp"`
		Group         string            `json:"group"`
		IsSegment     bool              `json:"is_segment"`
		Max           float64           `json:"max"`
		Min           float64           `json:"min"`
		Sum           float64           `json:"sum"`
		Mri           string            `json:"mri"`
		ProjectID     uint64            `json:"project_id"`
		Received      int64             `json:"received"`
		RetentionDays int               `json:"retention_days"`
		SegmentID     string            `json:"segment_id"`
		SpanID        string            `json:"span_id"`
		Tags          map[string]string `json:"tags"`
		TraceID       string            `json:"trace_id"`
	}
)

func buildFunctionsKafkaMessage(p profile.Profile, functions []nodetree.CallTreeFunction) FunctionsKafkaMessage {
	return FunctionsKafkaMessage{
		Environment:     p.Environment(),
		Functions:       functions,
		ID:              p.ID(),
		Platform:        p.Platform(),
		ProjectID:       p.ProjectID(),
		Received:        p.Received().Unix(),
		Release:         p.Release(),
		RetentionDays:   p.RetentionDays(),
		Timestamp:       p.Timestamp().Unix(),
		TransactionName: p.Transaction().Name,
	}
}

func buildProfileKafkaMessage(p profile.Profile) ProfileKafkaMessage {
	t := p.Transaction()
	m := p.Metadata()
	return ProfileKafkaMessage{
		AndroidAPILevel:      m.AndroidAPILevel,
		Architecture:         m.Architecture,
		DeviceClassification: m.DeviceClassification,
		DeviceLocale:         m.DeviceLocale,
		DeviceManufacturer:   m.DeviceManufacturer,
		DeviceModel:          m.DeviceModel,
		DeviceOSBuildNumber:  m.DeviceOSBuildNumber,
		DeviceOSName:         m.DeviceOSName,
		DeviceOSVersion:      m.DeviceOSVersion,
		DurationNS:           p.DurationNS(),
		Environment:          p.Environment(),
		ID:                   p.ID(),
		OrganizationID:       p.OrganizationID(),
		Platform:             p.Platform(),
		ProjectID:            p.ProjectID(),
		Received:             p.Received().Unix(),
		RetentionDays:        p.RetentionDays(),
		SDKName:              m.SDKName,
		SDKVersion:           m.SDKVersion,
		TraceID:              t.TraceID,
		TransactionID:        t.ID,
		TransactionName:      t.Name,
		VersionCode:          m.VersionCode,
		VersionName:          m.VersionName,
	}
}

type KafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}
