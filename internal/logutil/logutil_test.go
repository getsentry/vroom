package logutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/getsentry/vroom/internal/testutil"
	"github.com/google/go-cmp/cmp"
)

type recordingHandler struct {
	slog.Handler
	level   slog.Level
	err     error
	records []slog.Record
	ctx     context.Context
}

func (h *recordingHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *recordingHandler) Handle(ctx context.Context, record slog.Record) error {
	h.ctx = ctx
	h.records = append(h.records, record)
	return h.err
}

func TestTeeHandlerEnabled(t *testing.T) {
	for _, tt := range []struct {
		name   string
		levels []slog.Level
		want   bool
	}{
		{name: "empty"},
		{name: "all disabled", levels: []slog.Level{slog.LevelWarn, slog.LevelError}},
		{name: "first enabled", levels: []slog.Level{slog.LevelInfo, slog.LevelError}, want: true},
		{name: "last enabled", levels: []slog.Level{slog.LevelError, slog.LevelInfo}, want: true},
		{name: "all enabled", levels: []slog.Level{slog.LevelDebug, slog.LevelInfo}, want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var h teeHandler
			for _, level := range tt.levels {
				h.handlers = append(h.handlers, &recordingHandler{level: level})
			}
			if got := h.Enabled(t.Context(), slog.LevelInfo); got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTeeHandlerHandle(t *testing.T) {
	firstErr := errors.New("first handler failed")
	secondErr := errors.New("second handler failed")
	disabledErr := errors.New("disabled handler called")
	for _, tt := range []struct {
		name     string
		handlers []*recordingHandler
		wantErrs []error
	}{
		{name: "empty"},
		{name: "all enabled", handlers: []*recordingHandler{{}, {}}},
		{name: "disabled child", handlers: []*recordingHandler{{}, {level: slog.LevelError, err: disabledErr}}},
		{name: "all disabled", handlers: []*recordingHandler{{level: slog.LevelWarn, err: disabledErr}, {level: slog.LevelError, err: disabledErr}}},
		{name: "continues after error", handlers: []*recordingHandler{{err: firstErr}, {}}, wantErrs: []error{firstErr}},
		{name: "joins errors", handlers: []*recordingHandler{{err: firstErr}, {err: secondErr}}, wantErrs: []error{firstErr, secondErr}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var h teeHandler
			for _, child := range tt.handlers {
				h.handlers = append(h.handlers, child)
			}
			ctx := t.Context()
			record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
			record.AddAttrs(slog.String("key", "value"))
			err := h.Handle(ctx, record)
			if len(tt.wantErrs) == 0 && err != nil {
				t.Errorf("Handle() = %v, want nil", err)
			}
			for _, want := range tt.wantErrs {
				if !errors.Is(err, want) {
					t.Errorf("Handle() = %v, want error containing %v", err, want)
				}
			}
			for i, child := range tt.handlers {
				var want []slog.Record
				if child.Enabled(ctx, record.Level) {
					want = []slog.Record{record}
					if child.ctx != ctx {
						t.Errorf("child %d received a different context", i)
					}
				}
				if diff := testutil.Diff(want, child.records, cmp.AllowUnexported(slog.Record{}, slog.Value{})); diff != "" {
					t.Errorf("child %d records mismatch (-want +got):\n%s", i, diff)
				}
			}
		})
	}
}

func TestTeeHandlerWithAttrsAndGroup(t *testing.T) {
	var buffers [2]bytes.Buffer
	h := teeHandler{handlers: []slog.Handler{
		slog.NewJSONHandler(&buffers[0], nil),
		slog.NewJSONHandler(&buffers[1], nil),
	}}
	derived := h.WithAttrs([]slog.Attr{slog.String("outer", "value")}).WithGroup("group").WithAttrs([]slog.Attr{slog.Int("inner", 42)})
	record := slog.NewRecord(time.Time{}, slog.LevelInfo, "derived", 0)
	record.AddAttrs(slog.Bool("record", true))
	if err := derived.Handle(t.Context(), record); err != nil {
		t.Fatal(err)
	}
	if err := h.Handle(t.Context(), slog.NewRecord(time.Time{}, slog.LevelInfo, "original", 0)); err != nil {
		t.Fatal(err)
	}
	for i := range buffers {
		decoder := json.NewDecoder(&buffers[i])
		for _, want := range []map[string]any{
			{"level": "INFO", "msg": "derived", "outer": "value", "group": map[string]any{"inner": float64(42), "record": true}},
			{"level": "INFO", "msg": "original"},
		} {
			var got map[string]any
			if err := decoder.Decode(&got); err != nil {
				t.Fatal(err)
			}
			if diff := testutil.Diff(want, got); diff != "" {
				t.Errorf("child %d output mismatch (-want +got):\n%s", i, diff)
			}
		}
	}
}
