package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/julienschmidt/httprouter"

	"github.com/getsentry/vroom/internal/flamegraph"
	"github.com/getsentry/vroom/internal/snubautil"
)

const (
	minNumWorkers int           = 5
	timeout       time.Duration = time.Second * 5
)

func (env *environment) getFlamegraph(w http.ResponseWriter, r *http.Request) {
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

	urlValues := r.URL.Query()
	urlValues.Set("project_id", rawProjectID)
	r.URL.RawQuery = urlValues.Encode()

	hub.Scope().SetTag("project_id", rawProjectID)

	sqb, err := env.profilesQueryBuilderFromRequest(ctx, r.URL.Query(), organizationID)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s := sentry.StartSpan(ctx, "snuba.read")
	profileIDs, err := snubautil.GetProfileIDs(organizationID, 100, sqb)
	if err != nil {
		s.Finish()
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hub.Scope().SetTag("fetched_profiles", strconv.Itoa(len(profileIDs)))
	s.Finish()

	s = sentry.StartSpan(ctx, "processing")
	speedscope, err := flamegraph.GetFlamegraphFromProfiles(ctx, env.storage, organizationID, projectID, profileIDs, nil, minNumWorkers, timeout)
	if err != nil {
		s.Finish()
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.Finish()

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()
	b, err := json.Marshal(speedscope)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

type postFlamegraphFromProfileIDs struct {
	ProfileIDs []string `json:"profile_ids"`
	// Spans is optional. If not nil,
	// then at Span[i] we'll find a
	// list of span intervals for the
	// profile ProfileIDs[i]
	Spans *[][]flamegraph.SpanInterval `json:"spans,omitempty"`
}

func (env *environment) postFlamegraphFromProfileIDs(w http.ResponseWriter, r *http.Request) {
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

	var profiles postFlamegraphFromProfileIDs
	s := sentry.StartSpan(ctx, "processing")
	s.Description = "Decoding data"
	err = json.NewDecoder(r.Body).Decode(&profiles)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if profiles.Spans != nil && (len(*profiles.Spans) != len(profiles.ProfileIDs)) {
		hub.CaptureException(errors.New("flamegraph: lengths of profile_ids and spans don't match"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var numWorkers = getFlamegraphNumWorkers(len(profiles.ProfileIDs), minNumWorkers)

	s = sentry.StartSpan(ctx, "processing")
	speedscope, err := flamegraph.GetFlamegraphFromProfiles(ctx, env.storage, organizationID, projectID, profiles.ProfileIDs, profiles.Spans, numWorkers, timeout)
	if err != nil {
		s.Finish()
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.Finish()

	hub.Scope().SetTag("sent_profiles", strconv.Itoa(len(profiles.ProfileIDs)))

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()
	b, err := json.Marshal(speedscope)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
