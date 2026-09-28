package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   int
	}{
		{name: "healthy", status: http.StatusOK, want: 0},
		{name: "draining", status: http.StatusBadGateway, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()
			if got := checkHealth(srv.URL); got != tt.want {
				t.Errorf("checkHealth() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCheckHealthUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	if got := checkHealth(url); got != 1 {
		t.Errorf("checkHealth() = %d, want 1", got)
	}
}
