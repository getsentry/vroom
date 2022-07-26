package snubautil

import (
	"encoding/json"
	"time"

	"github.com/getsentry/sentry-go"
)

type (
	ProfilesStats struct {
		Time  time.Time `json:"time"`
		P75   float64   `json:"p75(),omitempty"`
		P99   float64   `json:"p99(),omitempty"`
		Count uint64    `json:"count(),omitempty"`
	}

	SnubaProfilesStatsResponse struct {
		Stats []ProfilesStats `json:"data"`
	}
)

func GetProfilesStats(sqb QueryBuilder) ([]ProfilesStats, error) {
	t := sentry.TransactionFromContext(sqb.ctx)
	rs := t.StartChild("snuba")
	defer rs.Finish()

	sqb.SelectCols = []string{
		"quantile(0.75)(duration_ns) AS `p75()`",
		"quantile(0.99)(duration_ns) AS `p99()`",
		"count() AS `count()`",
	}
	sqb.GroupBy = "time"
	sqb.OrderBy = "time ASC"
	sqb.Limit = 10000

	rb, err := sqb.Do(rs)
	if err != nil {
		return nil, err
	}
	defer rb.Close()

	var sr SnubaProfilesStatsResponse
	err = json.NewDecoder(rb).Decode(&sr)
	if err != nil {
		return nil, err
	}

	return sr.Stats, nil
}
