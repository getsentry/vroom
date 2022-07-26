package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/httputil"
	"github.com/getsentry/vroom/internal/snubautil"
	"github.com/julienschmidt/httprouter"
)

type (
	getProfilesStatsResponse struct {
		Data       []StatsData     `json:"data"`
		Meta       StatsMeta       `json:"meta"`
		Timestamps StatsTimestamps `json:"timestamps"`
	}

	StatsMeta struct {
		Dataset string `json:"dataset"`
		Start   int64  `json:"start"`
		End     int64  `json:"end"`
	}

	StatsTimestamps []int64

	StatsData struct {
		Values []*float64 `json:"values"`
		Axis   string     `json:"axis"`
	}
)

func (env *environment) getProfilesStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)

	p, ok := httputil.GetRequiredQueryParameters(w, r, "project_id", "start", "end", "granularity")
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	meta, err := getStatsMeta(p["start"], p["end"])
	if err != nil {
		hub.CaptureException(err)
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

	rawStats, err := snubautil.GetProfilesStats(sqb)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	timestamps, data := formatStats(rawStats, meta, int64(sqb.Granularity))

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

const TIME_LAYOUT = "2006-01-02T15:04:05.000000+00:00"

func getStatsMeta(startStr, endStr string) (StatsMeta, error) {
	start, err := time.Parse(TIME_LAYOUT, startStr)
	if err != nil {
		return StatsMeta{}, err
	}

	end, err := time.Parse(TIME_LAYOUT, endStr)
	if err != nil {
		return StatsMeta{}, err
	}

	return StatsMeta{
		Dataset: "profiles",
		Start:   start.Unix(),
		End:     end.Unix(),
	}, nil
}

func formatStats(rawStats []snubautil.ProfilesStats, meta StatsMeta, granularity int64) (StatsTimestamps, []StatsData) {
	start := meta.Start / granularity * granularity
	end := meta.End / granularity * granularity

	n := (end-start)/granularity + 1
	timestamps := make([]int64, n, n)

	countData := StatsData{
		Values: make([]*float64, n, n),
		Axis:   "count()",
	}

	p75Data := StatsData{
		Values: make([]*float64, n, n),
		Axis:   "p75()",
	}

	p99Data := StatsData{
		Values: make([]*float64, n, n),
		Axis:   "p99()",
	}

	rawIdx := 0

	for i, timestamp := 0, start; timestamp <= end; i, timestamp = i+1, timestamp+granularity {
		timestamps[i] = timestamp

		var count *float64 = nil
		var p75 *float64 = nil
		var p99 *float64 = nil

		if rawIdx < len(rawStats) && rawStats[rawIdx].Time.Unix() == timestamp {
			floatCount := float64(rawStats[rawIdx].Count)
			count = &floatCount
			p75 = &rawStats[rawIdx].P75
			p99 = &rawStats[rawIdx].P99
			rawIdx += 1
		}

		countData.Values[i] = count
		p75Data.Values[i] = p75
		p99Data.Values[i] = p99
	}

	return timestamps, []StatsData{countData, p75Data, p99Data}
}
