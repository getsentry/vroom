package storageutil

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/pierrec/lz4/v4"
)

// ErrObjectNotFound indicates an object was not found.
var ErrObjectNotFound = errors.New("object not found")

type ReadSizeCloser interface {
	io.Reader
	io.Closer
	Size() int64
}

// ObjectHandler provides common interface for multiple storage providers.
type ObjectHandler interface {
	// Put writes a file to the storage provider with name being the path.
	Put(ctx context.Context, name string) (io.WriteCloser, error)
	// Get reads a file from the storage provider with name being the path.
	// If a key was not found, it will return ErrObjectNotFound.
	Get(ctx context.Context, name string) (ReadSizeCloser, error)
}

// CompressedWrite compresses and writes data to Google Cloud Storage.
func CompressedWrite(ctx context.Context, b ObjectHandler, objectName string, d interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ow, err := b.Put(ctx, objectName)
	if err != nil {
		return err
	}
	zw := lz4.NewWriter(ow)
	_ = zw.Apply(lz4.CompressionLevelOption(lz4.Level9))
	jw := json.NewEncoder(zw)
	err = jw.Encode(d)
	if err != nil {
		return err
	}
	err = zw.Close()
	if err != nil {
		return err
	}
	err = ow.Close()
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalCompressed reads compressed JSON data from GCS and unmarshals it.
func UnmarshalCompressed(ctx context.Context, b ObjectHandler, objectName string, d interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	or, err := b.Get(ctx, objectName)
	if err != nil {
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
