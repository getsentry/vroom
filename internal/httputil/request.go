package httputil

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// GetRequiredQueryParameters attempts to read the specified query parameters
// from the request and returns a map of the key value pairs. If any of the required
// query parameters are missing or blank, it'll write a 400 status code as well as
// the reasoning for the error into the ResponseWriter, and also set return false.
func GetRequiredQueryParameters(w http.ResponseWriter, r *http.Request, paramKeys ...string) (map[string]string, zerolog.Logger, bool) {
	params := make(map[string]string, len(paramKeys))
	logger := log.With()
	for _, key := range paramKeys {
		value := r.URL.Query().Get(key)
		if value == "" {
			http.Error(w, fmt.Sprintf("expected %s query parameter", key), http.StatusBadRequest)
			return nil, zerolog.Nop(), false
		}
		params[key] = value
		logger = logger.Str(key, value)
	}
	return params, logger.Logger(), true
}
