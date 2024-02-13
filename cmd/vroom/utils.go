package main

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/rs/zerolog/log"

	"github.com/getsentry/vroom/internal/snubautil"
)

var (
	profileFilterFields = map[string]string{
		"device_classification":  "device_classification",
		"device_locale":          "device_locale",
		"device_manufacturer":    "device_manufacturer",
		"device_model":           "device_model",
		"device_os_build_number": "device_os_build_number",
		"device_os_name":         "device_os_name",
		"device_os_version":      "device_os_version",
		"environment":            "environment",
		"platform":               "platform",
		"transaction_id":         "transaction_id",
		"transaction_name":       "transaction_name",
	}

	profileQueryFilterMakers = []func(url.Values) ([]string, error){
		snubautil.MakeProjectsFilter,
		func(params url.Values) ([]string, error) {
			return snubautil.MakeTimeRangeFilter("received", params)
		},
		func(params url.Values) ([]string, error) {
			return snubautil.MakeFieldsFilter(profileFilterFields, params)
		},
		snubautil.MakeAndroidAPILevelFilter,
		snubautil.MakeVersionNameAndCodeFilter,
	}

	functionFilterFields = map[string]string{
		"device_os_name":    "os_name",
		"device_os_version": "os_version",
		"environment":       "environment",
		"platform":          "platform",
		"transaction_name":  "transaction_name",
		"version":           "release",
	}

	functionsQueryFilterMakers = []func(url.Values) ([]string, error){
		func(params url.Values) ([]string, error) {
			return snubautil.MakeTimeRangeFilter("timestamp", params)
		},
		snubautil.MakeApplicationFilter,
		func(params url.Values) ([]string, error) {
			return snubautil.MakeFieldsFilter(functionFilterFields, params)
		},
	}
)

func setExtrasFromRequest(sqb *snubautil.QueryBuilder, p url.Values) error {
	if v := p.Get("limit"); v != "" {
		limit, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			log.Err(err).Str("limit", v).Msg("can't parse limit value")
			return err
		}
		sqb.Limit = limit
	}

	if v := p.Get("offset"); v != "" {
		offset, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			log.Err(err).Str("offset", v).Msg("can't parse offset value")
			return err
		}
		sqb.Offset = offset
	}

	if v := p.Get("granularity"); v != "" {
		granularity, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			log.Err(err).Str("offset", v).Msg("can't parse granularity value")
			return err
		}
		sqb.Granularity = granularity
	}

	return nil
}

func (e *environment) profilesQueryBuilderFromRequest(
	ctx context.Context,
	p url.Values,
	orgID uint64,
) (snubautil.QueryBuilder, error) {
	sqb, err := e.snuba.NewQuery(ctx, "profiles", orgID)
	if err != nil {
		return snubautil.QueryBuilder{}, err
	}
	sqb.WhereConditions = make([]string, 0, 5)

	for _, makeFilter := range profileQueryFilterMakers {
		conditions, err := makeFilter(p)
		if err != nil {
			return snubautil.QueryBuilder{}, err
		}
		sqb.WhereConditions = append(sqb.WhereConditions, conditions...)
	}

	err = setExtrasFromRequest(&sqb, p)
	if err != nil {
		return snubautil.QueryBuilder{}, err
	}

	return sqb, nil
}

func (e *environment) functionsQueryBuilderFromRequest(
	ctx context.Context,
	p url.Values,
	orgID uint64,
) (snubautil.QueryBuilder, error) {
	sqb, err := e.snuba.NewQuery(ctx, "functions", orgID)
	if err != nil {
		return snubautil.QueryBuilder{}, err
	}
	sqb.WhereConditions = make([]string, 0, 5)

	// we do not want to show unknown functions, unknown package is okay
	sqb.WhereConditions = append(sqb.WhereConditions, "name != ''")

	for _, makeFilter := range functionsQueryFilterMakers {
		conditions, err := makeFilter(p)
		if err != nil {
			return snubautil.QueryBuilder{}, err
		}
		sqb.WhereConditions = append(sqb.WhereConditions, conditions...)
	}

	err = setExtrasFromRequest(&sqb, p)
	if err != nil {
		return snubautil.QueryBuilder{}, err
	}

	return sqb, nil
}

func getFlamegraphNumWorkers(numProfiles, minNumWorkers int) int {
	if numProfiles < minNumWorkers {
		return numProfiles
	}
	v := int(math.Ceil((float64(numProfiles) / 100) * float64(minNumWorkers)))
	return max(v, minNumWorkers)
}

func getSentryOptions(sentryHost string) (SentryOptions, error) {
	resp, err := http.Get(sentryHost + "/api/0/vrooms/options/")
	if err != nil {
		return SentryOptions{}, err
	}
	defer resp.Body.Close()

	var options SentryOptions
	err = json.NewDecoder(resp.Body).Decode(&options)
	if err != nil {
		return SentryOptions{}, err
	}
	return options, nil
}

type SentryOptions struct {
	ProfileMetricsSampleRate float32 `json:"profiling.profile_metrics.unsampled_profiles.sample_rate"`
}
