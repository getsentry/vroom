package main

type (
	ServiceConfig struct {
		Environment string `env:"SENTRY_ENVIRONMENT" env-default:"development"`
		Port        int    `env:"PORT"               env-default:"8085"`

		SentryDSN string `env:"SENTRY_DSN"`

		OccurrencesKafkaBrokers []string `env:"SENTRY_KAFKA_BROKERS_OCCURRENCES" env-default:"localhost:9092"`
		OccurrencesKafkaTopic   string   `env:"SENTRY_KAFKA_TOPIC_OCCURRENCES"   env-default:"ingest-occurrences"`

		ProfilingKafkaBrokers []string `env:"SENTRY_KAFKA_BROKERS_PROFILING" env-default:"localhost:9092"`
		CallTreesKafkaTopic   string   `env:"SENTRY_KAFKA_TOPIC_CALL_TREES"  env-default:"profiles-call-tree"`
		ProfilesKafkaTopic    string   `env:"SENTRY_KAKFA_TOPIC_PROFILES"    env-default:"processed-profiles"`

		SentryHost string `env:"SENTRY_SENTRY_HOST" env-default:"http://localhost:8000"`
		SnubaHost  string `env:"SENTRY_SNUBA_HOST" env-default:"http://localhost:1218"`

		BucketURL string `env:"SENTRY_BUCKET_PROFILES" env-default:"file://./test/gcs/sentry-profiles"`
	}
)
