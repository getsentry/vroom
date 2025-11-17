package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"google.golang.org/api/googleapi"

	"github.com/getsentry/vroom/internal/chunk"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/storageutil"
)

const (
	// when computing slowest functions, ignore frames/node whose depth in the callTree
	// is less than 1 (i.e. root frames).
	minDepth uint = 1
)

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
	Chunk         interface{} `json:"chunk"`
	DebugChunkIDs []string    `json:"debug_chunk_ids,omitempty"`
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
	go func() {
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
	}()

	chunkIDs := make([]string, 0, len(requestBody.ChunkIDs))
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
		fmt.Fprint(w, "error: no chunks found to merge")
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
			chunkIDs = append(chunkIDs, sc.ID)
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
		resp, err = json.Marshal(postProfileFromChunkIDsResponse{
			Chunk:         mergedChunk,
			DebugChunkIDs: chunkIDs,
		})
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
			chunkIDs = append(chunkIDs, ac.ID)
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
		resp, err = json.Marshal(postProfileFromChunkIDsResponse{
			Chunk:         sp,
			DebugChunkIDs: chunkIDs,
		})
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

func (env *environment) getRawChunk(w http.ResponseWriter, r *http.Request) {
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

	profilerID := ps.ByName("profiler_id")
	_, err = uuid.Parse(profilerID)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("profiler_id", profilerID)

	hub.Scope().SetTag("project_id", rawProjectID)

	chunkID := ps.ByName("chunk_id")
	_, err = uuid.Parse(chunkID)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("chunk_id", chunkID)

	s := sentry.StartSpan(ctx, "chunk.read")
	s.Description = "Read chunk from GCS"

	var c chunk.Chunk
	err = storageutil.UnmarshalCompressed(
		ctx,
		env.storage,
		chunk.StoragePath(organizationID, projectID, profilerID, chunkID),
		&c,
	)
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

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()
	b, err := json.Marshal(c)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600, immutable")
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
