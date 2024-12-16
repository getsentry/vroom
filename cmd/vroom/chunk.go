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
	"github.com/getsentry/vroom/internal/metrics"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/storageutil"
)

type (
	chunkPlatform struct {
		Platform platform.Platform `json:"platform"`
	}
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
	r.Body.Close()

	var p chunkPlatform
	err = json.Unmarshal(body, &p)
	if err != nil {
		if hub != nil {
			hub.CaptureException(err)
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var c chunk.Chunk
	switch p.Platform {
	case platform.Android:
		c = chunk.New(new(chunk.AndroidChunk))
	default:
		c = chunk.New(new(chunk.SampleChunk))
	}

	s = sentry.StartSpan(ctx, "json.unmarshal")
	s.Description = "Unmarshal profile"
	err = json.Unmarshal(body, &c)
	s.Finish()
	if err != nil {
		if hub != nil {
			hub.CaptureException(err)
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	c.Normalize()

	if hub != nil {
		hub.Scope().SetContext("Profile metadata", map[string]interface{}{
			"chunk_id":        c.GetID(),
			"organization_id": strconv.FormatUint(c.GetOrganizationID(), 10),
			"profiler_id":     c.GetProfilerID(),
			"project_id":      strconv.FormatUint(c.GetProjectID(), 10),
			"size":            len(body),
		})

		hub.Scope().SetTags(map[string]string{
			"platform": string(c.GetPlatform()),
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
			if code := gcerrors.Code(err); code == gcerrors.FailedPrecondition {
				w.WriteHeader(http.StatusPreconditionFailed)
			} else {
				if hub != nil {
					hub.CaptureException(err)
				}
				// These errors won't be retried
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

	// nb.: here we don't have a specific thread ID, so we're going to ingest
	// functions metrics from all the thread.
	// That's ok as this data is not supposed to be transaction/span scoped,
	// plus, we'll only retain application frames, so much of the system functions
	// chaff will be dropped.
	s = sentry.StartSpan(ctx, "processing")
	callTrees, err := c.CallTrees(nil)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s = sentry.StartSpan(ctx, "processing")
	s.Description = "Extract functions"
	functions := metrics.ExtractFunctionsFromCallTrees(callTrees)
	functions = metrics.CapAndFilterFunctions(functions, maxUniqueFunctionsPerProfile, true)
	s.Finish()

	// This block writes into the functions dataset
	s = sentry.StartSpan(ctx, "json.marshal")
	s.Description = "Marshal functions Kafka message"
	b, err = json.Marshal(buildChunkFunctionsKafkaMessage(&c, functions))
	s.Finish()
	if err != nil {
		if hub != nil {
			hub.CaptureException(err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s = sentry.StartSpan(ctx, "processing")
	s.Description = "Send functions to Kafka"
	err = env.profilingWriter.WriteMessages(ctx, kafka.Message{
		Topic: env.config.CallTreesKafkaTopic,
		Value: b,
	})
	s.Finish()
	if hub != nil {
		hub.Scope().SetContext("Call functions payload", map[string]interface{}{
			"Size": len(b),
		})
	}
	if err != nil {
		if hub != nil {
			hub.CaptureException(err)
		}
	}

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
	Chunk chunk.SampleChunk `json:"chunk"`
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
	r.Body.Close()

	hub.Scope().SetTag("num_chunks", fmt.Sprintf("%d", len(requestBody.ChunkIDs)))
	s = sentry.StartSpan(ctx, "chunks.read")
	s.Description = "Read profile chunks from GCS"

	results := make(chan storageutil.ReadJobResult, len(requestBody.ChunkIDs))
	defer close(results)
	// send a task to the workers pool for each chunk
	for _, ID := range requestBody.ChunkIDs {
		readJobs <- chunk.ReadJob{
			Ctx:            ctx,
			Storage:        env.storage,
			OrganizationID: organizationID,
			ProjectID:      projectID,
			ProfilerID:     requestBody.ProfilerID,
			ChunkID:        ID,
			Result:         results,
		}
	}

	chunks := make([]chunk.Chunk, 0, len(requestBody.ChunkIDs))
	// read the output of each tasks
	for i := 0; i < len(requestBody.ChunkIDs); i++ {
		res := <-results
		result, ok := res.(chunk.ReadJobResult)
		if !ok {
			continue
		}
		// if there was an error we assign it to the global error
		// so that we can later handle the response appropriately
		// and then we skip
		if result.Err != nil {
			err = result.Err
			continue
		} else if err != nil {
			// if this specific chunk download did not produce an error,
			// but a previous one did, we also skip since it doesn't make
			// sense to have a final profile with missing chunks
			continue
		}
		chunks = append(chunks, *result.Chunk)
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
	if len(chunks) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var resp []byte
	// Here we check what type of chunks we're dealing with,
	// since Android chunks and Sample chunks return completely
	// different types (Chunk vs Speedscope), hence we can't hide
	// the implementation behind an interface.
	//
	// We check the first chunk type, and use that to assert the
	// type of all the elements in the slice and then call the
	// appropriate utility.
	switch chunks[0].Chunk().(type) {
	case *chunk.SampleChunk:
		sampleChunks := make([]chunk.SampleChunk, 0, len(chunks))
		for _, c := range chunks {
			sc, ok := c.Chunk().(*chunk.SampleChunk)
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, "error: mix of sampled and android chunks")
				return
			}
			sampleChunks = append(sampleChunks, *sc)
		}
		mergedChunk, err := chunk.MergeSampleChunks(sampleChunks, requestBody.Start, requestBody.End)
		s.Finish()
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		s = sentry.StartSpan(ctx, "json.marshal")
		resp, err = json.Marshal(postProfileFromChunkIDsResponse{Chunk: mergedChunk})
		s.Finish()
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	case *chunk.AndroidChunk:
		androidChunks := make([]chunk.AndroidChunk, 0, len(chunks))
		for _, c := range chunks {
			ac, ok := c.Chunk().(*chunk.AndroidChunk)
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, "error: mix of android and sample chunks")
				return
			}
			androidChunks = append(androidChunks, *ac)
		}
		sp, err := chunk.SpeedscopeFromAndroidChunks(androidChunks, requestBody.Start, requestBody.End)
		s.Finish()
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		s = sentry.StartSpan(ctx, "json.marshal")
		resp, err = json.Marshal(sp)
		s.Finish()
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		// Should never happen.
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

type (
	ChunkKafkaMessage struct {
		ProjectID  uint64 `json:"project_id"`
		ProfilerID string `json:"profiler_id"`
		ChunkID    string `json:"chunk_id"`

		StartTimestamp float64 `json:"start_timestamp"`
		EndTimestamp   float64 `json:"end_timestamp"`
		DurationMS     uint64  `json:"duration_ms"`

		Received      float64 `json:"received"`
		RetentionDays int     `json:"retention_days"`

		Environment string            `json:"environment"`
		Platform    platform.Platform `json:"platform"`
		Release     string            `json:"release"`
		SDKName     string            `json:"sdk_name"`
		SDKVersion  string            `json:"sdk_version"`
	}
)

func buildChunkKafkaMessage(c chunk.Chunk) *ChunkKafkaMessage {
	return &ChunkKafkaMessage{
		ChunkID:        c.GetID(),
		DurationMS:     c.DurationMS(),
		EndTimestamp:   c.EndTimestamp(),
		Environment:    c.GetEnvironment(),
		Platform:       c.GetPlatform(),
		ProfilerID:     c.GetProfilerID(),
		ProjectID:      c.GetProjectID(),
		Received:       c.GetReceived(),
		Release:        c.GetRelease(),
		RetentionDays:  c.GetRetentionDays(),
		SDKName:        c.SDKName(),
		SDKVersion:     c.SDKVersion(),
		StartTimestamp: c.StartTimestamp(),
	}
}
