package storageutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/pierrec/lz4/v4"
	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"
	"google.golang.org/api/googleapi"
)

// ErrObjectNotFound indicates an object was not found.
var ErrObjectNotFound = errors.New("object not found")

// CompressedWrite compresses and writes data to Google Cloud Storage.
func CompressedWrite(ctx context.Context, b *blob.Bucket, objectName string, d interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	writerOptions := &blob.WriterOptions{IfNotExist: true}
	ow, err := b.NewWriter(ctx, objectName, writerOptions)
	if err != nil {
		return err
	}
	zw := lz4.NewWriter(ow)
	_ = zw.Apply(lz4.CompressionLevelOption(lz4.Level9))
	jw := json.NewEncoder(zw)
	err = jw.Encode(d)
	if err != nil {
		cancel()
		ow.Close()
		return err
	}
	err = zw.Close()
	if err != nil {
		cancel()
		ow.Close()
		return err
	}
	return ow.Close()
}

// UnmarshalCompressed reads compressed JSON data from GCS and unmarshals it.
func UnmarshalCompressed(
	ctx context.Context,
	b *blob.Bucket,
	objectName string,
	d interface{},
) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	or, err := b.NewReader(ctx, objectName, nil)
	if err != nil {
		// gocloud.dev returns 403s as 404s. Unwrap so we can see the difference.
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == http.StatusForbidden {
			return err
		}
		if errors.Is(err, gcerrors.ErrNotFound) {
			return fmt.Errorf("%w: %s", ErrObjectNotFound, objectName)
		}

		return err
	}
	defer or.Close()
	zr := lz4.NewReader(or)
	err = json.NewDecoder(zr).Decode(d)
	if err != nil {
		return err
	}
	return nil
}

type (
	ReadJob interface {
		Read()
	}

	ReadJobResult interface {
		Error() error
	}
)

func ReadWorker(jobs <-chan ReadJob) {
	for job := range jobs {
		job.Read()
	}
}
