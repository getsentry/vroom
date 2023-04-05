package main

import (
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
)

type (
	// FunctionsKafkaMessage is representing the struct we send to Kafka to insert functions in ClickHouse.
	FunctionsKafkaMessage struct {
		CallTrees       map[uint64][]*nodetree.Node `json:"call_trees"`
		Environment     string                      `json:"environment,omitempty"`
		ID              string                      `json:"profile_id"`
		OSName          string                      `json:"os_name"`
		OSVersion       string                      `json:"os_version"`
		Platform        platform.Platform           `json:"platform"`
		ProjectID       uint64                      `json:"project_id"`
		Received        int64                       `json:"received"`
		Release         string                      `json:"release"`
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
		TraceID              string            `json:"trace_id"`
		TransactionID        string            `json:"transaction_id"`
		TransactionName      string            `json:"transaction_name"`
		VersionCode          string            `json:"version_code"`
		VersionName          string            `json:"version_name"`
	}
)

func buildFunctionsKafkaMessage(p profile.Profile, functions []nodetree.CallTreeFunction) FunctionsKafkaMessage {
	// now we transform the results back into a manually constructed
	// node tree so it's compatible with snuba
	nodes := make([]*nodetree.Node, 0, len(functions))

	for _, function := range functions {
		nodes = append(nodes, function.ToNodes()...)
	}

	callTrees := map[uint64][]*nodetree.Node{
		// the tid can be anything since we dont use it anywhere
		0: nodes,
	}

	m := p.Metadata()
	return FunctionsKafkaMessage{
		CallTrees:       callTrees,
		Environment:     p.Environment(),
		ID:              p.ID(),
		OSName:          m.DeviceOSName,
		OSVersion:       m.DeviceOSVersion,
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
		TraceID:              t.TraceID,
		TransactionID:        t.ID,
		TransactionName:      t.Name,
		VersionCode:          m.VersionCode,
		VersionName:          m.VersionName,
	}
}
