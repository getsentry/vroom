package main

type (
	ServiceConfig struct {
		Environment string

		SentryDSN string

		OccurrencesEnabledOrganizations map[uint64]struct{}
		OccurrencesKafkaBrokers         []string
		OccurrencesKafkaTopic           string

		ProfilingKafkaBrokers []string
		CallTreesKafkaTopic   string
		ProfilesKafkaTopic    string

		ProfilesBucket string
		SnubaHost      string
	}
)

var (
	serviceConfigs = map[string]ServiceConfig{
		"production": {
			SentryDSN:               "https://91f2762536314cbd9cc4a163fe072682@o1.ingest.sentry.io/6424467",
			ProfilesBucket:          "sentry-profiles",
			SnubaHost:               "http://127.0.0.1:10006",
			OccurrencesKafkaTopic:   "ingest-occurrences",
			OccurrencesKafkaBrokers: []string{"kafka-issue-platform.service.us-central1.consul:9092"},
			ProfilingKafkaBrokers:   []string{"specto-dev-kafka.service.us-central1.consul:9092"},
			CallTreesKafkaTopic:     "profiles-call-tree",
			ProfilesKafkaTopic:      "processed-profiles",
		},
		"development": {
			ProfilesBucket:          "sentry-profiles",
			SnubaHost:               "http://localhost:1218",
			OccurrencesKafkaTopic:   "ingest-occurrences",
			OccurrencesKafkaBrokers: []string{"localhost:9092"},
			ProfilingKafkaBrokers:   []string{"localhost:9092"},
			CallTreesKafkaTopic:     "profiles-call-tree",
			ProfilesKafkaTopic:      "processed-profiles",
		},
	}
)
