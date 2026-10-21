package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/occurrence"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/storageutil"
	"github.com/segmentio/kafka-go"
)

type regressedKafkaWriter struct {
	calls    int
	messages []kafka.Message
	err      error
}

func (w *regressedKafkaWriter) WriteMessages(_ context.Context, messages ...kafka.Message) error {
	w.calls++
	w.messages = append(w.messages, messages...)
	return w.err
}

func (w *regressedKafkaWriter) Close() error {
	return nil
}

func TestPostRegressed(t *testing.T) {
	p := profile.New(&sample.Profile{
		Platform: platform.Node,
		Trace:    sample.Trace{Frames: []frame.Frame{{Function: "work"}}}})
	originalJobs := readJobs
	jobs := make(chan storageutil.ReadJob)
	readJobs = jobs
	done := make(chan struct{})
	go func() {
		defer close(done)
		for job := range jobs {
			job.(profile.ReadJob).Result <- profile.ReadJobResult{Profile: &p}
		}
	}()
	t.Cleanup(func() {
		close(jobs)
		<-done
		readJobs = originalJobs
	})

	const payload = `[{"organization_id":1,"project_id":2,"profile_id":"test","fingerprint":1176630088}]`
	for _, tt := range []struct {
		name       string
		query      string
		body       string
		writerErr  error
		wantCalls  int
		wantCount  int
		wantInline bool
		wantStatus int
	}{
		{name: "default", body: payload, wantCalls: 1, wantCount: 1, wantStatus: http.StatusOK},
		{name: "empty", query: "?produce=", body: payload, wantCalls: 1, wantCount: 1, wantStatus: http.StatusOK},
		{name: "bare", query: "?produce", body: payload, wantCalls: 1, wantCount: 1, wantStatus: http.StatusOK},
		{name: "true", query: "?produce=true", body: payload, wantCalls: 1, wantCount: 1, wantStatus: http.StatusOK},
		{name: "zero", query: "?produce=0", body: payload, wantCalls: 1, wantCount: 1, wantStatus: http.StatusOK},
		{name: "uppercase", query: "?produce=False", body: payload, wantCalls: 1, wantCount: 1, wantStatus: http.StatusOK},
		{name: "false", query: "?produce=false", body: payload, wantInline: true, wantCount: 1, wantStatus: http.StatusOK},
		{name: "false empty batch", query: "?produce=false", body: "[]", wantInline: true, wantStatus: http.StatusOK},
		{name: "writer failure", body: payload, writerErr: errors.New("write failed"), wantCalls: 1, wantStatus: http.StatusInternalServerError},
		{name: "false ignores writer failure", query: "?produce=false", body: payload, writerErr: errors.New("write failed"), wantInline: true, wantCount: 1, wantStatus: http.StatusOK},
		{name: "invalid payload", query: "?produce=false", body: "invalid", wantStatus: http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writer := &regressedKafkaWriter{err: tt.writerErr}
			env := environment{occurrencesWriter: writer}
			rec := httptest.NewRecorder()
			env.postRegressed(rec, httptest.NewRequest(http.MethodPost, "/regressed"+tt.query, strings.NewReader(tt.body)))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if writer.calls != tt.wantCalls {
				t.Fatalf("Kafka calls = %d, want %d", writer.calls, tt.wantCalls)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			var response struct {
				Occurrences        int                            `json:"occurrences"`
				Emitted            []occurrence.RegressedFunction `json:"emitted"`
				OccurrencePayloads []*occurrence.Occurrence       `json:"occurrence_payloads"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Occurrences != tt.wantCount || len(response.Emitted) != tt.wantCount {
				t.Fatalf("occurrences = %d, emitted = %d, want %d", response.Occurrences, len(response.Emitted), tt.wantCount)
			}
			if (response.OccurrencePayloads != nil) != tt.wantInline {
				t.Fatalf("occurrence_payloads presence mismatch: %s", rec.Body.String())
			}
			var occurrences []*occurrence.Occurrence
			if tt.wantInline {
				occurrences = response.OccurrencePayloads
			} else {
				for _, message := range writer.messages {
					var o occurrence.Occurrence
					if err := json.Unmarshal(message.Value, &o); err != nil {
						t.Fatal(err)
					}
					occurrences = append(occurrences, &o)
				}
			}
			if len(occurrences) != tt.wantCount {
				t.Fatalf("payload count = %d, want %d", len(occurrences), tt.wantCount)
			}
			for _, o := range occurrences {
				if o.ProjectID != 2 || o.Culprit != "work" || o.ID == "" || o.PayloadType != occurrence.OccurrencePayload {
					t.Fatalf("unexpected occurrence: %+v", o)
				}
			}
		})
	}
}
