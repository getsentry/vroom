package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func useTempDownFile(t *testing.T) {
	t.Helper()
	orig := downFile
	downFile = filepath.Join(t.TempDir(), "vroom.down")
	t.Cleanup(func() { downFile = orig })
}

func healthStatus(t *testing.T) int {
	t.Helper()
	var e environment
	rec := httptest.NewRecorder()
	e.getHealth(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	return rec.Code
}

func TestDrain(t *testing.T) {
	useTempDownFile(t)

	if got := healthStatus(t); got != http.StatusOK {
		t.Fatalf("health before drain = %d, want %d", got, http.StatusOK)
	}
	if got := runDrain([]string{"0s"}); got != 0 {
		t.Fatalf("runDrain() = %d, want 0", got)
	}
	if got := healthStatus(t); got != http.StatusBadGateway {
		t.Fatalf("health after drain = %d, want %d", got, http.StatusBadGateway)
	}

	clearDownFile()
	if got := healthStatus(t); got != http.StatusOK {
		t.Fatalf("health after clearing = %d, want %d", got, http.StatusOK)
	}
}

func TestDrainInvalidArgs(t *testing.T) {
	useTempDownFile(t)

	for _, args := range [][]string{nil, {"25s", "extra"}, {"25"}, {"-1s"}} {
		if got := runDrain(args); got != 1 {
			t.Errorf("runDrain(%q) = %d, want 1", args, got)
		}
	}
	if _, err := os.Stat(downFile); err == nil {
		t.Error("down file created despite invalid arguments")
	}
}
