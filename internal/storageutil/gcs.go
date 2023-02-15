package storageutil

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/storage"
	"github.com/pierrec/lz4/v4"
)

// CompressedWrite compresses and writes data to Google Cloud Storage.
func CompressedWrite(ctx context.Context, b *storage.BucketHandle, objectName string, d interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ow := b.Object(objectName).NewWriter(ctx)
	zw := lz4.NewWriter(ow)
	_ = zw.Apply(lz4.CompressionLevelOption(lz4.Level9))
	jw := json.NewEncoder(zw)
	err := jw.Encode(d)
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
func UnmarshalCompressed(ctx context.Context, b *storage.BucketHandle, objectName string, d interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	or, err := b.Object(objectName).NewReader(ctx)
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
