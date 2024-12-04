package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/julienschmidt/httprouter"

	"github.com/getsentry/vroom/internal/flamegraph"
	"github.com/getsentry/vroom/internal/metrics"
	"github.com/getsentry/vroom/internal/utils"
)

type (
	postFlamegraphBody struct {
		Transaction     []utils.TransactionProfileCandidate `json:"transaction"`
		Continuous      []utils.ContinuousProfileCandidate  `json:"continuous"`
		GenerateMetrics bool                                `json:"generate_metrics"`
	}
)

func (env *environment) postFlamegraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)
	ps := httprouter.ParamsFromContext(ctx)
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		if hub != nil {
			hub.CaptureException(err)
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("organization_id", rawOrganizationID)

	var body postFlamegraphBody
	s := sentry.StartSpan(ctx, "processing")
	s.Description = "Decoding data"
	err = json.NewDecoder(r.Body).Decode(&body)
	s.Finish()
	if err != nil {
		if hub != nil {
			hub.CaptureException(err)
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s = sentry.StartSpan(ctx, "processing")
	var ma *metrics.Aggregator
	if body.GenerateMetrics {
		agg := metrics.NewAggregator(maxUniqueFunctionsPerProfile, 5)
		ma = &agg
	}
	speedscope, err := flamegraph.GetFlamegraphFromCandidates(
		ctx,
		env.storage,
		organizationID,
		body.Transaction,
		body.Continuous,
		readJobs,
		ma,
		s,
	)
	s.Finish()
	if err != nil {
		if hub != nil {
			hub.CaptureException(err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()
	b, err := json.Marshal(speedscope)
	if err != nil {
		if hub != nil {
			hub.CaptureException(err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
