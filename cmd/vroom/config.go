package main

type (
	ServiceConfig struct {
		Environment string `env:"SENTRY_ENVIRONMENT" env-default:"development"`
		Port        int    `env:"PORT" env-default:"8085"`

		SentryDSN string `env:"SENTRY_DSN"`

		OccurrencesKafkaBrokers []string `env:"SENTRY_KAFKA_BROKERS_OCCURRENCES" env-default:"localhost:9092"`
		OccurrencesKafkaTopic   string   `env:"SENTRY_KAFKA_TOPIC_OCCURRENCES" env-default:"ingest-occurrences"`

		ProfilingKafkaBrokers []string `env:"SENTRY_KAFKA_BROKERS_PROFILING" env-default:"localhost:9092"`
		CallTreesKafkaTopic   string   `env:"SENTRY_KAFKA_TOPIC_CALL_TREES" env-default:"profiles-call-tree"`
		ProfilesKafkaTopic    string   `env:"SENTRY_KAKFA_TOPIC_PROFILES" env-default:"processed-profiles"`

		SnubaHost string `env:"SENTRY_SNUBA_HOST" env-default:"http://localhost:1218"`

		ProfilesStorageProvider string `env:"SENTRY_PROFILES_STORAGE_PROVIDER" env-default:"gcs"`
		GCSProfileBucket        string `env:"SENTRY_GCS_BUCKET_PROFILES" env-default:"sentry-profiles"`
		BadgerDBProfilePath     string `env:"SENTRY_BADGERDB_PROFILE_PATH" env-default:"/var/lib/badgerdb"`
	}
)
