package httputil

import (
	"io"
	"net/http"

	"github.com/andybalholm/brotli"
)

// DecompressPayload adds a reader of the right type in case you need to decompress the body
func DecompressPayload(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Header.Get("Content-Encoding") == "br" {
			r.Body = io.NopCloser(brotli.NewReader(r.Body))
		}

		next.ServeHTTP(w, r)
	})
}
