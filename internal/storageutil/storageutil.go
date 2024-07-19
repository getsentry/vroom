package storageutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/storage"
	"github.com/pierrec/lz4/v4"
	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"
)

// ErrObjectNotFound indicates an object was not found.
var ErrObjectNotFound = errors.New("object not found")

// CompressedWrite compresses and writes data to Google Cloud Storage.
func CompressedWrite(ctx context.Context, b *blob.Bucket, objectName string, d interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	writerOptions := &blob.WriterOptions{
		BeforeWrite: func(asFunc func(interface{}) bool) error {
			var objp **storage.ObjectHandle
			// If it's not a GCS resource, we just move on.
			if !asFunc(&objp) {
				return nil
			}
			// Replace the ObjectHandle with a new one that adds Conditions.
			*objp = (*objp).If(storage.Conditions{DoesNotExist: true})
			return nil
		},
	}
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
		if gcerrors.Code(err) == gcerrors.NotFound {
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
