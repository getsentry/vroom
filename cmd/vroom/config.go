package main

type (
	ServiceConfig struct {
		Environment    string `env:"SENTRY_ENVIRONMENT" env-default:"development"`
		Port           int    `env:"PORT"               env-default:"8085"`
		WorkerPoolSize int    `env:"WORKER_POOL_SIZE"               env-default:"10"`

		SentryDSN string `env:"SENTRY_DSN"`

		KafkaSaslMechanism string `env:"SENTRY_KAFKA_SASL_MECHANISM"`
		KafkaSaslUsername  string `env:"SENTRY_KAFKA_SASL_USERNAME"`
		KafkaSaslPassword  string `env:"SENTRY_KAFKA_SASL_PASSWORD"`
		KafkaSslCaPath     string `env:"SENTRY_KAFKA_SSL_CA_PATH"`
		KafkaSslCertPath   string `env:"SENTRY_KAFKA_SSL_CERT_PATH"`
		KafkaSslKeyPath    string `env:"SENTRY_KAFKA_SSL_KEY_PATH"`

		OccurrencesKafkaBrokers []string `env:"SENTRY_KAFKA_BROKERS_OCCURRENCES" env-default:"localhost:9092"`

		OccurrencesKafkaTopic string `env:"SENTRY_KAFKA_TOPIC_OCCURRENCES" env-default:"ingest-occurrences"`

		BucketURL string `env:"SENTRY_BUCKET_PROFILES" env-default:"file://./test/gcs/sentry-profiles"`
	}
)
