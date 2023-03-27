package logutil

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"cloud.google.com/go/compute/metadata"
)

func ConfigureLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.With().Caller().Stack().Logger()
	if metadata.OnGCE() {
		log.Logger = log.Hook(ErrorHook{})
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}

type ErrorHook struct{}

func (h ErrorHook) Run(e *zerolog.Event, level zerolog.Level, _ string) {
	e.Str("severity", level.String())
}
