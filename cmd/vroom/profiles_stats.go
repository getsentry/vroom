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
		Data       []snubautil.StatsData `json:"data"`
		Meta       snubautil.StatsMeta   `json:"meta"`
		Timestamps []snubautil.UnixTime  `json:"timestamps"`
	}
)

func (env *environment) getProfilesStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)

	p, ok := httputil.GetRequiredQueryParameters(w, r, "project_id", "start", "end", "granularity")
	if !ok {
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

	rawStats, err := snubautil.GetProfilesStats(sqb)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	meta, err := snubautil.NewStatsMeta("profiles", p["start"], p["end"], sqb.Granularity)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data, timestamps := snubautil.FormatStats(meta, rawStats, []string{"p75", "p99", "count"})

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

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
