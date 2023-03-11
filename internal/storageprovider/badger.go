package storageprovider

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/dgraph-io/badger/v4"
	"github.com/getsentry/vroom/internal/storageutil"
)

// Badger implements storageutil.ObjectHandler interface to handle object read and writes.
type Badger struct {
	DB *badger.DB
}

// Put writes a file to the storage provider with name being the path.
func (b *Badger) Put(ctx context.Context, name string) (io.WriteCloser, error) {
	transaction := b.DB.NewTransaction(true)
	return &badgerWriter{
		b:    &bytes.Buffer{},
		txn:  transaction,
		name: name,
	}, nil
}

// Get reads a file from the storage provider with name being the path.
// If a key was not found, it will return ErrObjectNotFound.
func (b *Badger) Get(ctx context.Context, name string) (storageutil.ReadSizeCloser, error) {
	transaction := b.DB.NewTransaction(false)
	item, err := transaction.Get([]byte(name))
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, storageutil.ErrObjectNotFound
		}

		return nil, err
	}

	value, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return &badgerReader{
		txn:    transaction,
		reader: bytes.NewReader(value),
		size:   item.ValueSize(),
	}, nil
}

// badgerWriter implements io.WriteCloser
type badgerWriter struct {
	b    *bytes.Buffer
	txn  *badger.Txn
	name string
}

func (bw *badgerWriter) Write(b []byte) (n int, err error) {
	n, err = bw.Write(b)
	if err != nil {
		bw.txn.Discard()
	}

	return
}

func (bw *badgerWriter) Close() error {
	err := bw.txn.Set([]byte(bw.name), bw.b.Bytes())
	if err != nil {
		return err
	}

	return bw.txn.Commit()
}

// badgerReader implements storageutil.ReadSizeCloser
type badgerReader struct {
	txn    *badger.Txn
	reader io.Reader
	size   int64
}

func (b *badgerReader) Read(p []byte) (n int, err error) {
	return b.reader.Read(p)
}

func (b *badgerReader) Close() error {
	return b.txn.Commit()
}

func (b *badgerReader) Size() int64 {
	return b.size
}
