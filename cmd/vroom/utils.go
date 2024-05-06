package main

import (
	"context"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
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

func extractMetricsFromFunctions(p *profile.Profile, functions []nodetree.CallTreeFunction) ([]sentry.Metric, []MetricSummary) {
	metrics := make([]sentry.Metric, 0, len(functions))
	metricsSummary := make([]MetricSummary, 0, len(functions))

	for _, function := range functions {
		if len(function.SelfTimesNS) == 0 {
			continue
		}
		tags := map[string]string{
			"project_id":     strconv.FormatUint(p.ProjectID(), 10),
			"transaction":    p.Transaction().Name,
			"fingerprint":    strconv.FormatUint(uint64(function.Fingerprint), 10),
			"name":           function.Function,
			"package":        function.Package,
			"is_application": strconv.FormatBool(function.InApp),
			"platform":       string(p.Platform()),
			"environment":    p.Environment(),
			"release":        p.Release(),
			"os_name":        p.Metadata().DeviceOSName,
			"os_version":     p.Metadata().DeviceOSVersion,
		}
		duration := float64(function.SelfTimesNS[0] / 1e6)
		summary := MetricSummary{
			Min:   duration,
			Max:   duration,
			Sum:   duration,
			Count: 1,
		}
		dm := sentry.NewDistributionMetric("profiles/function.duration", sentry.MilliSecond(), tags, p.Metadata().Timestamp, duration)
		// loop remaining selfTime durations
		for i := 1; i < len(function.SelfTimesNS); i++ {
			duration := float64(function.SelfTimesNS[i] / 1e6)
			dm.Add(duration)
			summary.Min = min(summary.Min, duration)
			summary.Max = max(summary.Max, duration)
			summary.Sum = summary.Sum + duration
			summary.Count = summary.Count + 1
		}
		metrics = append(metrics, dm)
		metricsSummary = append(metricsSummary, summary)
	}
	return metrics, metricsSummary
}

func sendMetrics(p *profile.Profile, metrics []sentry.Metric, mClient *http.Client) {
	id := strings.Replace(uuid.New().String(), "-", "", -1)
	e := sentry.NewEvent()
	e.EventID = sentry.EventID(id)
	e.Type = "statsd"
	e.Metrics = metrics
	tr := sentry.NewHTTPSyncTransport()
	tr.Timeout = 5 * time.Second
	tr.Configure(sentry.ClientOptions{
		Dsn:           p.GetOptions().ProjectDSN,
		HTTPTransport: mClient.Transport,
		HTTPClient:    mClient,
	})

	tr.SendEvent(e)
}

type MetricSummary struct {
	Min   float64
	Max   float64
	Sum   float64
	Count uint64
}
