package storageutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"gocloud.dev/blob/gcsblob"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type gcsErrorTransport struct {
	statusCode int
}

func (tr gcsErrorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body := fmt.Sprintf(`{"error":{"code":%d,"message":%q}}`, tr.statusCode, http.StatusText(tr.statusCode))
	return &http.Response{
		StatusCode: tr.statusCode,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func TestUnmarshalCompressedGCSErrors(t *testing.T) {
	for _, statusCode := range []int{http.StatusForbidden, http.StatusNotFound} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			ctx := context.Background()
			client, err := storage.NewClient(ctx, option.WithHTTPClient(&http.Client{
				Transport: gcsErrorTransport{statusCode: statusCode},
			}))
			if err != nil {
				t.Fatal(err)
			}
			bucket, err := gcsblob.OpenBucket(ctx, nil, "profiles", &gcsblob.Options{Client: client})
			if err != nil {
				_ = client.Close()
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := bucket.Close(); err != nil {
					t.Error(err)
				}
			})

			var profile Profile
			err = UnmarshalCompressed(ctx, bucket, "1/2/profile", &profile)
			if err == nil {
				t.Fatal("expected a storage error")
			}
			if statusCode == http.StatusNotFound {
				if !errors.Is(err, ErrObjectNotFound) {
					t.Fatalf("expected ErrObjectNotFound, got %v", err)
				}
				return
			}
			if errors.Is(err, ErrObjectNotFound) {
				t.Fatalf("permission error must not become ErrObjectNotFound: %v", err)
			}
			var apiErr *googleapi.Error
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected the underlying Google API error to be preserved, got %v", err)
			}
			if apiErr.Code != statusCode {
				t.Errorf("expected HTTP status %d, got %d", statusCode, apiErr.Code)
			}
		})
	}
}
