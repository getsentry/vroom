package main

type (
	ServiceConfig struct {
		Environment    string `env:"SENTRY_ENVIRONMENT" env-default:"development"`
		Port           int    `env:"PORT"               env-default:"8085"`
		WorkerPoolSize int    `env:"WORKER_POOL_SIZE"               env-default:"25"`

		SentryDSN string `env:"SENTRY_DSN"`

		KafkaSaslMechanism string `env:"SENTRY_KAFKA_SASL_MECHANISM"`
		KafkaSaslUsername  string `env:"SENTRY_KAFKA_SASL_USERNAME"`
		KafkaSaslPassword  string `env:"SENTRY_KAFKA_SASL_PASSWORD"`
		KafkaSslCaPath     string `env:"SENTRY_KAFKA_SSL_CA_PATH"`
		KafkaSslCertPath   string `env:"SENTRY_KAFKA_SSL_CERT_PATH"`
		KafkaSslKeyPath    string `env:"SENTRY_KAFKA_SSL_KEY_PATH"`

		OccurrencesKafkaBrokers []string `env:"SENTRY_KAFKA_BROKERS_OCCURRENCES" env-default:"localhost:9092"`
		ProfilingKafkaBrokers   []string `env:"SENTRY_KAFKA_BROKERS_PROFILING" env-default:"localhost:9092"`
		SpansKafkaBrokers       []string `env:"SENTRY_KAFKA_BROKERS_SPANS" env-default:"localhost:9092"`

		CallTreesKafkaTopic     string `env:"SENTRY_KAFKA_TOPIC_CALL_TREES" env-default:"profiles-call-tree"`
		OccurrencesKafkaTopic   string `env:"SENTRY_KAFKA_TOPIC_OCCURRENCES" env-default:"ingest-occurrences"`
		ProfileChunksKafkaTopic string `env:"SENTRY_KAFKA_TOPIC_PROFILE_CHUNKS" env-default:"snuba-profile-chunks"`
		ProfilesKafkaTopic      string `env:"SENTRY_KAKFA_TOPIC_PROFILES" env-default:"processed-profiles"`

		SnubaHost string `env:"SENTRY_SNUBA_HOST" env-default:"http://localhost:1218"`

		BucketURL string `env:"SENTRY_BUCKET_PROFILES" env-default:"file://./test/gcs/sentry-profiles"`
	}
)
