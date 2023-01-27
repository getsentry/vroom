package main

import (
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
)

type (
	ProfileKafkaMessage struct {
		CallTrees       map[uint64][]*nodetree.Node `json:"call_trees,omitempty"`
		Environment     string                      `json:"environment,omitempty"`
		ID              string                      `json:"profile_id"`
		OSName          string                      `json:"os_name"`
		OSVersion       string                      `json:"os_version"`
		Platform        platform.Platform           `json:"platform"`
		ProjectID       uint64                      `json:"project_id"`
		Release         string                      `json:"release"`
		RetentionDays   int                         `json:"retention_days,omitempty"`
		Timestamp       int64                       `json:"timestamp"`
		TransactionName string                      `json:"transaction_name"`
	}
)

func buildProfileKafkaMessage(p profile.Profile, callTrees map[uint64][]*nodetree.Node) ProfileKafkaMessage {
	return ProfileKafkaMessage{
		CallTrees:       callTrees,
		Environment:     p.Environment(),
		ID:              p.ID(),
		OSName:          p.OSName(),
		OSVersion:       p.OSVersion(),
		Platform:        p.Platform(),
		ProjectID:       p.ProjectID(),
		Release:         p.Release(),
		Timestamp:       p.Timestamp().Unix(),
		TransactionName: p.Transaction().Name,
	}
}
