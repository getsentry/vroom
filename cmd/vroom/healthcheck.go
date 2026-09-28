package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// runHealthcheck queries the local /health endpoint and returns the process
// exit code. It lets container healthchecks run without a shell in the image.
func runHealthcheck() int {
	port := 8085
	if p := os.Getenv("PORT"); p != "" {
		var err error
		port, err = strconv.Atoi(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid PORT %q: %v\n", p, err)
			return 1
		}
	}
	return checkHealth(fmt.Sprintf("http://127.0.0.1:%d/health", port))
}

func checkHealth(url string) int {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck failed: status %d\n", resp.StatusCode)
		return 1
	}
	return 0
}
