package main

import (
	"encoding/json"
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
)

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
		wantCount  int
		wantStatus int
	}{
		{name: "default", body: payload, wantCount: 1, wantStatus: http.StatusOK},
		{name: "legacy false ignored", query: "?produce=false", body: payload, wantCount: 1, wantStatus: http.StatusOK},
		{name: "legacy true ignored", query: "?produce=true", body: payload, wantCount: 1, wantStatus: http.StatusOK},
		{name: "empty batch", body: "[]", wantStatus: http.StatusOK},
		{name: "null batch", body: "null", wantStatus: http.StatusOK},
		{name: "missing frame", body: strings.Replace(payload, "1176630088", "0", 1), wantStatus: http.StatusOK},
		{name: "invalid payload", body: "invalid", wantStatus: http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env := environment{}
			rec := httptest.NewRecorder()
			env.postRegressed(rec, httptest.NewRequest(http.MethodPost, "/regressed"+tt.query, strings.NewReader(tt.body)))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
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
			if response.OccurrencePayloads == nil || response.Emitted == nil {
				t.Fatalf("expected response arrays, got: %s", rec.Body.String())
			}
			if len(response.OccurrencePayloads) != tt.wantCount {
				t.Fatalf("payload count = %d, want %d", len(response.OccurrencePayloads), tt.wantCount)
			}
			for _, o := range response.OccurrencePayloads {
				if o.ProjectID != 2 || o.Culprit != "work" || o.ID == "" || o.PayloadType != occurrence.OccurrencePayload {
					t.Fatalf("unexpected occurrence: %+v", o)
				}
			}
		})
	}
}
