package storageprovider

import (
	"context"
	"errors"
	"io"

	"cloud.google.com/go/storage"
	"github.com/getsentry/vroom/internal/storageutil"
)

// Gcs implements storageutil.ObjectHandler interface to handle object read and writes.
type Gcs struct {
	BucketHandle *storage.BucketHandle
}

// Put writes a file to the storage provider with name being the path.
func (g *Gcs) Put(ctx context.Context, name string) (io.WriteCloser, error) {
	return g.BucketHandle.Object(name).NewWriter(ctx), nil
}

// Get reads a file from the storage provider with name being the path.
// If a key was not found, it will return ErrObjectNotFound.
func (g *Gcs) Get(ctx context.Context, name string) (storageutil.ReadSizeCloser, error) {
	rc, err := g.BucketHandle.Object(name).NewReader(ctx)
	if err != nil && errors.Is(err, storage.ErrObjectNotExist) {
		return nil, storageutil.ErrObjectNotFound
	}

	return rc, err
}
