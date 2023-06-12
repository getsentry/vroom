package httputil

import (
	"strconv"

	"github.com/getsentry/sentry-go"
)

// HTTPStatusCodeTag is the name of the HTTP status code tag.
const HTTPStatusCodeTag = "http.response.status_code"

// SetHTTPStatusCodeTag sets the status code tag for the current request to the top-level transaction.
// TODO: Move this to the SDK itself.
func SetHTTPStatusCodeTag(e *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	if hint.Response == nil {
		return e
	}
	if e.Tags == nil {
		e.Tags = make(map[string]string)
	}
	if _, exists := e.Tags[HTTPStatusCodeTag]; !exists {
		e.Tags[HTTPStatusCodeTag] = strconv.Itoa(hint.Response.StatusCode)
	}
	return e
}
