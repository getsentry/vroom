package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/httputil"
	"github.com/getsentry/vroom/internal/snubautil"
	"github.com/julienschmidt/httprouter"
)

type (
	getProfilesStatsResponse struct {
		Data       []snubautil.StatsData     `json:"data"`
		Meta       snubautil.StatsMeta       `json:"meta"`
		Timestamps snubautil.StatsTimestamps `json:"timestamps"`
	}

	RawProfilesStats struct {
		data []snubautil.ProfilesStats
	}
)

func (s RawProfilesStats) Axes() []string {
	return []string{"count()", "p75()", "p99()"}
}

func (s RawProfilesStats) TimestampAt(idx int) int64 {
	if idx >= len(s.data) {
		return -1
	}
	return s.data[idx].Time.Unix()
}

func (s RawProfilesStats) ValueAt(axis string, idx int) (float64, error) {
	switch axis {
	case "count()":
		return float64(s.data[idx].Count), nil
	case "p75()":
		return s.data[idx].P75, nil
	case "p99()":
		return s.data[idx].P99, nil
	default:
		return 0, fmt.Errorf("unknown axis: %s", axis)
	}
}

func (env *environment) getProfilesStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)

	p, ok := httputil.GetRequiredQueryParameters(w, r, "project_id", "start", "end", "granularity")
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("project_id", p["project_id"])

	ps := httprouter.ParamsFromContext(ctx)

	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("organization_id", rawOrganizationID)

	sqb, err := env.profilesQueryBuilderFromRequest(ctx, r.URL.Query())
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sqb.WhereConditions = append(sqb.WhereConditions,
		fmt.Sprintf("organization_id=%d", organizationID),
	)

	meta, err := snubautil.FormatStatsMeta("profiles", p["start"], p["end"], int64(sqb.Granularity))
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rawStats, err := snubautil.GetProfilesStats(sqb)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	timestamps, data, err := snubautil.FormatStats(RawProfilesStats{data: rawStats}, meta)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(getProfilesStatsResponse{
		Data:       data,
		Meta:       meta,
		Timestamps: timestamps,
	})
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
	return
}
