package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

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

func createKafkaRoundTripper(e ServiceConfig) kafka.RoundTripper {
	var saslMechanism sasl.Mechanism = nil
	var tlsConfig *tls.Config = nil

	switch strings.ToUpper(e.KafkaSaslMechanism) {
	case "PLAIN":
		saslMechanism = plain.Mechanism{
			Username: e.KafkaSaslUsername,
			Password: e.KafkaSaslPassword,
		}
	case "SCRAM-SHA-256":
		mechanism, err := scram.Mechanism(scram.SHA256, e.KafkaSaslUsername, e.KafkaSaslPassword)
		if err != nil {
			log.Fatal("unable to create scram-sha-256 mechanism", err)
			return nil
		}

		saslMechanism = mechanism
	case "SCRAM-SHA-512":
		mechanism, err := scram.Mechanism(scram.SHA512, e.KafkaSaslUsername, e.KafkaSaslPassword)
		if err != nil {
			log.Fatal("unable to create scram-sha-512 mechanism", err)
			return nil
		}

		saslMechanism = mechanism
	}

	if e.KafkaSslCertPath != "" && e.KafkaSslKeyPath != "" {
		certs, err := tls.LoadX509KeyPair(e.KafkaSslCertPath, e.KafkaSslKeyPath)
		if err != nil {
			log.Fatal("unable to load certificate key pair", err)
			return nil
		}

		caCertificatePool, err := x509.SystemCertPool()
		if err != nil {
			caCertificatePool = x509.NewCertPool()
		}
		if e.KafkaSslCaPath != "" {
			caFile, err := os.ReadFile(e.KafkaSslCaPath)
			if err != nil {
				log.Fatal("unable to read ca file", err)
				return nil
			}

			if ok := caCertificatePool.AppendCertsFromPEM(caFile); !ok {
				log.Fatal("unable to append ca certificate to pool")
				return nil
			}
		}

		tlsConfig = &tls.Config{
			RootCAs:      caCertificatePool,
			Certificates: []tls.Certificate{certs},
		}
	}

	return &kafka.Transport{
		SASL: saslMechanism,
		TLS:  tlsConfig,
	}
}
