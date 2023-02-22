package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/julienschmidt/httprouter"

	"github.com/getsentry/vroom/internal/flamegraph"
	"github.com/getsentry/vroom/internal/snubautil"
)

func (env *environment) getFlamegraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)
	const numWorkers int = 5
	const timeout time.Duration = time.Second * 5
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

	sqb, err := env.profilesQueryBuilderFromRequest(ctx, r.URL.Query())
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
	s.Finish()

	s = sentry.StartSpan(ctx, "processing")
	speedscope, err := flamegraph.GetFlamegraphFromProfiles(ctx, env.profilesBucket, organizationID, projectID, profileIDs, numWorkers, timeout)
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
