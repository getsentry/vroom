package logutil

import (
	"log/slog"
	"os"
)

func ConfigureLogger() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
}
