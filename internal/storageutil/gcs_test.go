package storageutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/google/uuid"
	"github.com/phayes/freeport"
	"github.com/pierrec/lz4/v4"
)

const bucketName = "profiles"

var server *fakestorage.Server

type Profile struct {
	Samples []int `json:"samples"`
	Frames  []int `json:"frames"`
}

func TestMain(m *testing.M) {
	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatalf("no free port found: %v", err)
	}
	publicHost := fmt.Sprintf("127.0.0.1:%d", port)
	server, err = fakestorage.NewServerWithOptions(fakestorage.Options{
		PublicHost: publicHost,
		Host:       "127.0.0.1",
		Port:       uint16(port),
		Scheme:     "http",
	})
	if err != nil {
		log.Fatalf("couldn't set up gcs server: %v", err)
	}
	os.Setenv("STORAGE_EMULATOR_HOST", publicHost)
	server.CreateBucketWithOpts(fakestorage.CreateBucketOpts{Name: bucketName})

	code := m.Run()
	os.Exit(code)
}

func TestUploadProfile(t *testing.T) {
	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatalf("we should be able to create a client: %v", err)
	}
	bucket := storageClient.Bucket(bucketName)
	objectName := uuid.New().String()
	originalData := []byte(`{"samples": [1, 2, 3, 4], "frames": [1, 2, 3, 4]}`)
	_, err = CompressedWrite(ctx, bucket, objectName, originalData)
	if err != nil {
		t.Fatalf("we should be able to write: %v", err)
	}
	object, err := server.GetObject(bucketName, objectName)
	if err != nil {
		t.Fatalf("we should be able to read the object: %v", err)
	}
	r := lz4.NewReader(bytes.NewBuffer(object.Content))
	uncompressedData, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("we should be able to uncompress the data: %v", err)
	}
	if !bytes.Equal(originalData, uncompressedData) {
		t.Fatal("data should be identical")
	}
}

func TestDownloadProfile(t *testing.T) {
	originalData := []byte(`{"samples":[1,2,3,4],"frames":[1,2,3,4]}`)
	var compressedData bytes.Buffer
	w := lz4.NewWriter(&compressedData)
	_, _ = w.Write(originalData)
	err := w.Close()
	if err != nil {
		t.Fatalf("we should be able to close the writer: %v", err)
	}
	objectName := uuid.New().String()

	server.CreateObject(fakestorage.Object{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: bucketName,
			Name:       objectName,
		},
		Content: compressedData.Bytes(),
	})

	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatalf("we should be able to create a client: %v", err)
	}
	bucket := storageClient.Bucket(bucketName)
	var profile Profile
	err = UnmarshalCompressed(ctx, bucket, objectName, &profile)
	if err != nil {
		t.Fatalf("we should be able to read the object: %v", err)
	}

	uncompressedData, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("we should be able to marshal back to JSON: %v", err)
	}
	if !bytes.Equal(originalData, uncompressedData) {
		t.Fatalf("data should be identical: %v %v", string(originalData), string(uncompressedData))
	}
}
