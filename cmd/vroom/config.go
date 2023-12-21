package main

type (
	ServiceConfig struct {
		Environment string `env:"SENTRY_ENVIRONMENT" yaml:"environment" env-default:"development"`
		Host        string `env:"HOST" yaml:"host"`
		Port        string `env:"PORT" yaml:"port" env-default:"8085"`

		SentryDSN string `env:"SENTRY_DSN" yaml:"sentry_dsn"`

		Occurrences struct {
			OccurrencesKafkaBrokers []string `env:"SENTRY_KAFKA_BROKERS_OCCURRENCES" yaml:"kafka_brokers" env-default:"localhost:9092"`
			OccurrencesKafkaTopic   string   `env:"SENTRY_KAFKA_TOPIC_OCCURRENCES"  yaml:"kafka_topic" env-default:"ingest-occurrences"`
		} `yaml:"occurrences"`

		Profiling struct {
			ProfilingKafkaBrokers []string `env:"SENTRY_KAFKA_BROKERS_PROFILING" yaml:"kafka_brokers" env-default:"localhost:9092"`
			CallTreesKafkaTopic   string   `env:"SENTRY_KAFKA_TOPIC_CALL_TREES" yaml:"call_trees_kafka_topic" env-default:"profiles-call-tree"`
			ProfilesKafkaTopic    string   `env:"SENTRY_KAKFA_TOPIC_PROFILES" yaml:"profiles_kafka_topic" env-default:"processed-profiles"`
		} `yaml:"profiling"`

		SnubaHost string `env:"SENTRY_SNUBA_HOST" yaml:"snuba_host" env-default:"http://localhost:1218"`

		BucketURL string `env:"SENTRY_BUCKET_PROFILES" yaml:"bucket_url" env-default:"file://./test/gcs/sentry-profiles"`

		Logging struct {
			Level  string `env:"SENTRY_LOGGING_LEVEL" yaml:"level" env-default:"info"`
			Format string `env:"SENTRY_LOGGING_FORMAT" yaml:"format" env-default:"simplified"`
		} `yaml:"logging"`
	}
)
