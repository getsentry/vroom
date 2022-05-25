package httputil

import (
	"fmt"
	"net/http"

	"github.com/getsentry/sentry-go"
)

// GetRequiredQueryParameters attempts to read the specified query parameters
// from the request and returns a map of the key value pairs. If any of the required
// query parameters are missing or blank, it'll write a 400 status code as well as
// the reasoning for the error into the ResponseWriter, and also set return false.
func GetRequiredQueryParameters(w http.ResponseWriter, r *http.Request, hub *sentry.Hub, keys ...string) (map[string]string, bool) {
	params := make(map[string]string, len(keys))
	context := make(map[string]interface{}, len(keys))
	for _, key := range keys {
		value := r.URL.Query().Get(key)
		if value == "" {
			http.Error(w, fmt.Sprintf("expected %s query parameter", key), http.StatusBadRequest)
			return nil, false
		}
		params[key] = value
		context[key] = value
	}
	return params, true
}
