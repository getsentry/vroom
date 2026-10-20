package logutil

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"slices"

	sentryslog "github.com/getsentry/sentry-go/slog"
)

// ConfigureLogger sets up logging to both stdout and Sentry. (Messages sent
// before Sentry is initialized will only be sent to stdout.)
func ConfigureLogger() {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			return a
		},
	})
	// `context.Background()` here will use the global Sentry client.
	sentryHandler := sentryslog.Option{
		LogLevel: []slog.Level{slog.LevelInfo, slog.LevelWarn, slog.LevelError, sentryslog.LevelFatal},
	}.NewSentryHandler(context.Background())
	h := teeHandler{handlers: []slog.Handler{jsonHandler, sentryHandler}}
	slog.SetDefault(slog.New(h))
	slog.SetLogLoggerLevel(slog.LevelInfo)
}

// teeHandler is a [slog.Handler] that forwards calls to all of its child
// handlers.
type teeHandler struct {
	handlers []slog.Handler
}

var _ slog.Handler = teeHandler{}

func (t teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return slices.ContainsFunc(t.handlers, func(h slog.Handler) bool {
		return h.Enabled(ctx, level)
	})
}

func (t teeHandler) Handle(ctx context.Context, record slog.Record) error {
	errs := make([]error, 0, len(t.handlers))
	for _, h := range t.handlers {
		if !h.Enabled(ctx, record.Level) {
			continue
		}
		if err := h.Handle(ctx, record); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (t teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(t.handlers))
	for i, h := range t.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &teeHandler{handlers: handlers}
}

func (t teeHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(t.handlers))
	for i, h := range t.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &teeHandler{handlers: handlers}
}
