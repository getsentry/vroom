package main

import (
	"context"
	"log"
	"os"
	"testing"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
)

var fileBlobBucket *blob.Bucket

func TestMain(m *testing.M) {
	temporaryDirectory, err := os.MkdirTemp(os.TempDir(), "sentry-profiles-*")
	if err != nil {
		log.Fatalf("couldn't create a temporary directory: %s", err.Error())
	}

	fileBlobBucket, err = blob.OpenBucket(context.Background(), "file://localhost/"+temporaryDirectory)
	if err != nil {
		log.Fatalf("couldn't open a local filesystem bucket: %s", err.Error())
	}

	code := m.Run()

	if err := fileBlobBucket.Close(); err != nil {
		log.Printf("couldn't close the local filesystem bucket: %s", err.Error())
	}

	err = os.RemoveAll(temporaryDirectory)
	if err != nil {
		log.Printf("couldn't remove the temporary directory: %s", err.Error())
	}

	os.Exit(code)
}
