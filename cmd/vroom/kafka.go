package main

import (
	"context"

	"github.com/getsentry/vroom/internal/chunk"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
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

func buildFunctionsKafkaMessage(p profile.Profile, functions []nodetree.CallTreeFunction) FunctionsKafkaMessage {
	return FunctionsKafkaMessage{
		Environment:            p.Environment(),
		Functions:              functions,
		ID:                     p.ID(),
		Platform:               p.Platform(),
		ProjectID:              p.ProjectID(),
		Received:               p.Received().Unix(),
		Release:                p.Release(),
		RetentionDays:          p.RetentionDays(),
		Timestamp:              p.Timestamp().Unix(),
		TransactionName:        p.Transaction().Name,
		MaterializationVersion: 1,
	}
}

// Metrics extraction is only supported for sample chunks right now.
// TODO: support metrics extraction for Android chunks.
func buildChunkFunctionsKafkaMessage(c *chunk.Chunk, functions []nodetree.CallTreeFunction) FunctionsKafkaMessage {
	return FunctionsKafkaMessage{
		Environment:            c.GetEnvironment(),
		Functions:              functions,
		ID:                     c.GetProfilerID(),
		Platform:               c.GetPlatform(),
		ProjectID:              c.GetProjectID(),
		Received:               int64(c.GetReceived()),
		Release:                c.GetRelease(),
		RetentionDays:          c.GetRetentionDays(),
		Timestamp:              int64(c.StartTimestamp()),
		StartTimestamp:         c.StartTimestamp(),
		EndTimestamp:           c.EndTimestamp(),
		ProfilingType:          "continuous",
		MaterializationVersion: 1,
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
