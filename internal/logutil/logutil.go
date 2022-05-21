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
	if !metadata.OnGCE() {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}
