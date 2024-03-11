package logutil

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"cloud.google.com/go/compute/metadata"
)

func ConfigureLogger(level string, format string) {
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.WarnLevel
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.With().Caller().Stack().Logger()
	log.Logger = log.Sample(LevelSampler{Level: logLevel})

	if metadata.OnGCE() {
		log.Logger = log.Hook(ErrorHook{})
	} else {
		if format == "simplified" {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		}
	}
}

type ErrorHook struct{}

func (h ErrorHook) Run(e *zerolog.Event, level zerolog.Level, _ string) {
	e.Str("severity", level.String())
}
