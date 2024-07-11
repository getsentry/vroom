package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/julienschmidt/httprouter"
	"github.com/segmentio/kafka-go"
	"gocloud.dev/gcerrors"
	"google.golang.org/api/googleapi"

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
		if hub != nil {
			hub.CaptureException(err)
		}
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
		if hub != nil {
			hub.CaptureException(err)
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if hub != nil {
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
	}

	s = sentry.StartSpan(ctx, "gcs.write")
	s.Description = "Write profile to GCS"
	err = storageutil.CompressedWrite(ctx, env.storage, c.StoragePath(), c)
	s.Finish()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			// This is a transient error, we'll retry
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			// These errors won't be retried
			if hub != nil {
				hub.CaptureException(err)
			}
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
		if hub != nil {
			hub.CaptureException(err)
		}
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
		if hub != nil {
			hub.CaptureException(err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.Finish()

	w.WriteHeader(http.StatusNoContent)
}

type postProfileFromChunkIDsRequest struct {
	ProfilerID string   `json:"profiler_id"`
	ChunkIDs   []string `json:"chunk_ids"`
	Start      uint64   `json:"start,string"`
	End        uint64   `json:"end,string"`
}

// Instead of returning Chunk directly, we'll return this struct
// that wraps a chunk.
// This way, if we decide to later add a few more utility fields
// (for pagination, etc.) we won't have to change the Chunk struct.
type postProfileFromChunkIDsResponse struct {
	Chunk chunk.Chunk `json:"chunk"`
}

// This is more of a GET method, but since we're receiving a list of chunk IDs as part of a
// body request, we use a POST method instead (similarly to the flamegraph endpoint).
func (env *environment) postProfileFromChunkIDs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)
	ps := httprouter.ParamsFromContext(ctx)
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("organization_id", rawOrganizationID)

	rawProjectID := ps.ByName("project_id")
	projectID, err := strconv.ParseUint(rawProjectID, 10, 64)
	if err != nil {
		sentry.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	hub.Scope().SetTag("project_id", rawProjectID)

	var requestBody postProfileFromChunkIDsRequest
	s := sentry.StartSpan(ctx, "processing")
	s.Description = "Decoding data"
	err = json.NewDecoder(r.Body).Decode(&requestBody)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("num_chunks", fmt.Sprintf("%d", len(requestBody.ChunkIDs)))
	s = sentry.StartSpan(ctx, "chunks.read")
	s.Description = "Read profile chunks from GCS"

	results := make(chan chunk.TaskOutput, len(requestBody.ChunkIDs))
	defer close(results)
	// send a task to the workers pool for each chunk
	for _, ID := range requestBody.ChunkIDs {
		jobs <- chunk.TaskInput{
			Ctx:            ctx,
			ProfilerID:     requestBody.ProfilerID,
			ChunkID:        ID,
			OrganizationID: organizationID,
			ProjectID:      projectID,
			Storage:        env.storage,
			Result:         results,
		}
	}

	chunks := make([]chunk.Chunk, 0, len(requestBody.ChunkIDs))
	// read the output of each tasks
	for i := 0; i < len(requestBody.ChunkIDs); i++ {
		res := <-results
		// if there was an error we assign it to the global error
		// so that we can later handle the response appropriately
		// and then we skip
		if res.Err != nil {
			err = res.Err
			continue
		} else if err != nil {
			// if this specific chunk download did not produce an error,
			// but a previous one did, we also skip since it doesn't make
			// sense to have a final profile with missing chunks
			continue
		}
		chunks = append(chunks, res.Chunk)
	}
	s.Finish()
	if err != nil {
		if errors.Is(err, storageutil.ErrObjectNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var e *googleapi.Error
		if ok := errors.As(err, &e); ok {
			hub.Scope().SetContext("Google Cloud Storage Error", map[string]interface{}{
				"body":    e.Body,
				"code":    e.Code,
				"details": e.Details,
				"message": e.Message,
			})
		}
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s = sentry.StartSpan(ctx, "chunks.merge")
	s.Description = "Merge profile chunks into a single one"
	chunk, err := chunk.MergeChunks(chunks, float64(requestBody.Start), float64(requestBody.End))
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()
	b, err := json.Marshal(postProfileFromChunkIDsResponse{Chunk: chunk})
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
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

// A worker download a chunk and send it back through
// the Result channel.
func worker(jobs <-chan chunk.TaskInput) {
	for task := range jobs {
		var c chunk.Chunk
		err := storageutil.UnmarshalCompressed(
			task.Ctx,
			task.Storage,
			chunk.StoragePath(task.OrganizationID, task.ProjectID, task.ProfilerID, task.ChunkID),
			&c,
		)
		task.Result <- chunk.TaskOutput{Chunk: c, Err: err}
	}
}
