package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/segmentio/kafka-go"
	"gocloud.dev/gcerrors"

	"github.com/getsentry/vroom/internal/chunk"
	"github.com/getsentry/vroom/internal/storageutil"
)

func (env *environment) postChunk(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)

	s := sentry.StartSpan(ctx, "processing")
	s.Description = "Read HTTP body"
	body, err := io.ReadAll(r.Body)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	c := new(chunk.Chunk)
	s = sentry.StartSpan(ctx, "json.unmarshal")
	s.Description = "Unmarshal profile"
	err = json.Unmarshal(body, c)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetContext("Profile metadata", map[string]interface{}{
		"chunk_id":        c.ID,
		"organization_id": strconv.FormatUint(c.OrganizationID, 10),
		"profiler_id":     c.ProfilerID,
		"project_id":      strconv.FormatUint(c.ProjectID, 10),
		"size":            len(body),
	})

	hub.Scope().SetTags(map[string]string{
		"platform": string(c.Platform),
	})

	s = sentry.StartSpan(ctx, "gcs.write")
	s.Description = "Write profile to GCS"
	err = storageutil.CompressedWrite(ctx, env.storage, c.StoragePath(), body)
	s.Finish()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			// This is a transient error, we'll retry
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			// These errors won't be retried
			hub.CaptureException(err)
			if code := gcerrors.Code(err); code == gcerrors.FailedPrecondition {
				w.WriteHeader(http.StatusPreconditionFailed)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
		return
	}

	s = sentry.StartSpan(ctx, "json.marshal")
	s.Description = "Marshal chunk Kafka message"
	b, err := json.Marshal(buildChunkKafkaMessage(c))
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s = sentry.StartSpan(ctx, "processing")
	s.Description = "Send chunk to Kafka"
	err = env.profilingWriter.WriteMessages(ctx, kafka.Message{
		Topic: env.config.ProfileChunksKafkaTopic,
		Value: b,
	})
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.Finish()

	w.WriteHeader(http.StatusNoContent)
}

type (
	ChunkKafkaMessage struct {
		ProjectID  uint64 `json:"project_id"`
		ProfilerID string `json:"profiler_id"`
		ChunkID    string `json:"chunk_id"`

		StartTimestamp float64 `json:"start_timestamp"`
		EndTimestamp   float64 `json:"end_timestamp"`

		Received      float64 `json:"received"`
		RetentionDays int     `json:"retention_days"`
	}
)

func buildChunkKafkaMessage(c *chunk.Chunk) *ChunkKafkaMessage {
	start, end := c.StartEndTimestamps()
	return &ChunkKafkaMessage{
		ChunkID:        c.ID,
		ProjectID:      c.ProjectID,
		ProfilerID:     c.ProfilerID,
		StartTimestamp: start,
		EndTimestamp:   end,
		Received:       c.Received,
		RetentionDays:  c.RetentionDays,
	}
}
