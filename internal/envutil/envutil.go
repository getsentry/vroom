package envutil

import (
	"fmt"
	"os"
	"runtime"

	"github.com/rs/zerolog/log"
)

// GetEnvOrError gets the environment variable for the specified key, and returns
// an error if the key is not found.
func GetEnvOrError(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("%q environment variable was not set", key)
	}
	return v, nil
}

// GetEnvOrFatal gets the environment variable for the specified key, and then
// logs the failure and terminates immediately if the key is not found.
func GetEnvOrFatal(key string) string {
	v := os.Getenv(key)
	_, file, line, _ := runtime.Caller(2)
	if v == "" {
		log.Fatal().Str("parent_caller_file", file).Int("parent_caller_line", line).Msg(key + " env var was not set")
	}
	return v
}

// GetPort returns the port number to bind to from the PORT environment variable,
// or the default port (8080) if it has not been set.
func GetPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "8080"
	}
	return port
}

// GetEnvOrFallback gets the environment variable for the specified key, but if
// it doesn't find a value, it'll instead return fallback.
func GetEnvOrFallback(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		value = fallback
	}
	return value
}
