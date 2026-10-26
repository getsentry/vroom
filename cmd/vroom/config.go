package main

type (
	ServiceConfig struct {
		Environment    string `env:"SENTRY_ENVIRONMENT" env-default:"development"`
		Port           int    `env:"PORT"               env-default:"8085"`
		WorkerPoolSize int    `env:"WORKER_POOL_SIZE"               env-default:"10"`

		SentryDSN string `env:"SENTRY_DSN"`

		BucketURL string `env:"SENTRY_BUCKET_PROFILES" env-default:"file://./test/gcs/sentry-profiles"`
	}
)
