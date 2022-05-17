package logutil

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"cloud.google.com/go/compute/metadata"
)

// ConfigureLogger performs one time logging configuration for the current execution
// environment.
func ConfigureLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.With().Caller().Stack().Logger()
	if !metadata.OnGCE() {
		// Running locally, not in GCP
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	} else {
		// Running inside GCP
		log.Logger = log.Hook(ErrorHook{})
	}
}

type ErrorHook struct{}

func (h ErrorHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	e.Str("severity", level.String())
	switch level {
	case zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel:
		e.Str("@type", "type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent")
	default:
		break
	}
}
