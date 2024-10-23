package main

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/chunk"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
)

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
			"fingerprint":    strconv.FormatUint(uint64(function.Fingerprint), 10),
			"function":       function.Function,
			"package":        function.Package,
			"is_application": strconv.FormatBool(function.InApp),
			"platform":       string(p.Platform()),
			"environment":    p.Environment(),
			"release":        p.Release(),
			"profile_type":   "transaction",
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

func sendMetrics(ctx context.Context, dsn string, metrics []sentry.Metric, mClient *http.Client) {
	id := strings.Replace(uuid.New().String(), "-", "", -1)
	e := sentry.NewEvent()
	e.EventID = sentry.EventID(id)
	e.Type = "statsd"
	e.Metrics = metrics
	tr := sentry.NewHTTPSyncTransport()
	tr.Timeout = 5 * time.Second
	tr.Configure(sentry.ClientOptions{
		Dsn:           dsn,
		HTTPTransport: mClient.Transport,
		HTTPClient:    mClient,
	})

	tr.SendEventWithContext(ctx, e)
}

type MetricSummary struct {
	Min   float64
	Max   float64
	Sum   float64
	Count uint64
}

func extractMetricsFromSampleChunkFunctions(c *chunk.SampleChunk, functions []nodetree.CallTreeFunction) []sentry.Metric {
	metrics := make([]sentry.Metric, 0, len(functions))

	for _, function := range functions {
		if len(function.SelfTimesNS) == 0 {
			continue
		}
		tags := map[string]string{
			"project_id":     strconv.FormatUint(c.ProjectID, 10),
			"fingerprint":    strconv.FormatUint(uint64(function.Fingerprint), 10),
			"function":       function.Function,
			"package":        function.Package,
			"is_application": strconv.FormatBool(function.InApp),
			"platform":       string(c.Platform),
			"environment":    c.Environment,
			"release":        c.Release,
			"profile_type":   "continuous",
		}
		duration := float64(function.SelfTimesNS[0] / 1e6)
		dm := sentry.NewDistributionMetric("profiles/function.duration", sentry.MilliSecond(), tags, int64(c.Received), duration)
		// loop remaining selfTime durations
		for i := 1; i < len(function.SelfTimesNS); i++ {
			duration := float64(function.SelfTimesNS[i] / 1e6)
			dm.Add(duration)
		}
		metrics = append(metrics, dm)
	}
	return metrics
}
