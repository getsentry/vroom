package httputil

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/julienschmidt/httprouter"
)

// GetRequiredQueryParameters attempts to read the specified query parameters
// from the request and returns a map of the key value pairs. If any of the required
// query parameters are missing or blank, it'll write a 400 status code as well as
// the reasoning for the error into the ResponseWriter, and also set return false.
func GetRequiredQueryParameters(w http.ResponseWriter, r *http.Request, keys ...string) (map[string]string, bool) {
	params := make(map[string]string, len(keys))
	for _, key := range keys {
		value := r.URL.Query().Get(key)
		if value == "" {
			http.Error(w, fmt.Sprintf("expected %s query parameter", key), http.StatusBadRequest)
			return nil, false
		}
		params[key] = value
	}
	return params, true
}

func AnonymizeTransactionName(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := httprouter.ParamsFromContext(r.Context())
		path := r.URL.Path
		for _, param := range params {
			path = strings.Replace(path, param.Value, fmt.Sprintf(":%s", param.Key), 1)
		}
		transaction := sentry.TransactionFromContext(r.Context())
		transaction.Name = fmt.Sprintf("%s %s", r.Method, path)

		handler.ServeHTTP(w, r)
	}
}
