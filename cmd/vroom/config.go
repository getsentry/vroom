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
		"production": ServiceConfig{
			SentryDSN:      "https://91f2762536314cbd9cc4a163fe072682@o1.ingest.sentry.io/6424467",
			ProfilesBucket: "sentry-profiles",
			SnubaHost:      "http://snuba-api.profiling",
			OccurrencesEnabledOrganizations: map[uint64]struct{}{
				1:      {},
				447951: {},
			},
			OccurrencesKafkaTopic:   "ingest-occurrences",
			OccurrencesKafkaBrokers: []string{"192.168.142.19:9092", "192.168.142.20:9092", "192.168.142.21:9092"},
			ProfilingKafkaBrokers:   []string{"192.168.142.19:9092", "192.168.142.20:9092", "192.168.142.21:9092"},
			CallTreesKafkaTopic:     "profiles-call-tree",
			ProfilesKafkaTopic:      "processed-profiles",
		},
		"development": ServiceConfig{
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
