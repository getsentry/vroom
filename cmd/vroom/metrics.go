package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/julienschmidt/httprouter"

	"github.com/getsentry/vroom/internal/metrics"
	"github.com/getsentry/vroom/internal/utils"
)

type (
	postMetricsRequestBody struct {
		Transaction []utils.TransactionProfileCandidate `json:"transaction"`
		Continuous  []utils.ContinuousProfileCandidate  `json:"continuous"`
	}

	postMetricsResponse struct {
		FunctionsMetrics []utils.FunctionMetrics `json:"functions_metrics"`
	}
)

func (env *environment) postMetrics(w http.ResponseWriter, r *http.Request) {
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

	var body postMetricsRequestBody
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
	ma := metrics.NewAggregator(maxUniqueFunctionsPerProfile, 5, minDepth)
	functionsMetrics, err := ma.GetMetricsFromCandidates(
		ctx,
		env.storage,
		organizationID,
		body.Transaction,
		body.Continuous,
		readJobs,
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
	b, err := json.Marshal(postMetricsResponse{
		FunctionsMetrics: functionsMetrics,
	})
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
