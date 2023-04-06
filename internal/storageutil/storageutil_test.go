package storageutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/google/uuid"
	"github.com/phayes/freeport"
	"github.com/pierrec/lz4/v4"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"

	gojson "github.com/goccy/go-json"
	jsoniter "github.com/json-iterator/go"
)

const bucketName = "profiles"

var gcsServer *fakestorage.Server
var gcsBlobBucket *blob.Bucket
var fileBlobBucket *blob.Bucket

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
	gcsServer, err = fakestorage.NewServerWithOptions(fakestorage.Options{
		PublicHost: publicHost,
		Host:       "127.0.0.1",
		Port:       uint16(port),
		Scheme:     "http",
	})
	if err != nil {
		log.Fatalf("couldn't set up gcs server: %v", err)
	}
	os.Setenv("STORAGE_EMULATOR_HOST", publicHost)
	gcsServer.CreateBucketWithOpts(fakestorage.CreateBucketOpts{Name: bucketName})

	temporaryDirectory, err := os.MkdirTemp(os.TempDir(), "sentry-profiles-*")
	if err != nil {
		log.Fatalf("couldn't create a temporary directory: %s", err.Error())
	}

	gcsBlobBucket, err = blob.OpenBucket(context.Background(), "gs://"+bucketName)
	if err != nil {
		log.Fatalf("couldn't open a local gcs bucket: %s", err.Error())
	}
	fileBlobBucket, err = blob.OpenBucket(context.Background(), "file://localhost/"+temporaryDirectory)
	if err != nil {
		log.Fatalf("couldn't open a local filesystem bucket: %s", err.Error())
	}

	code := m.Run()

	if err := gcsBlobBucket.Close(); err != nil {
		log.Printf("couldn't close the local gcs bucket: %s", err.Error())
	}

	if err := fileBlobBucket.Close(); err != nil {
		log.Printf("couldn't close the local filesystem bucket: %s", err.Error())
	}

	err = os.RemoveAll(temporaryDirectory)
	if err != nil {
		log.Printf("couldn't remove the temporary directory: %s", err.Error())
	}

	gcsServer.Stop()

	os.Exit(code)
}

func TestUploadProfile(t *testing.T) {
	ctx := context.Background()
	objectName := uuid.New().String()
	originalData := struct {
		Samples []uint64 `json:"samples"`
		Frames  []uint64 `json:"frames"`
	}{
		Samples: []uint64{1, 2, 3, 4},
		Frames:  []uint64{1, 2, 3, 4},
	}

	tests := []struct {
		name       string
		blobBucket *blob.Bucket
	}{
		{
			name:       "GCS",
			blobBucket: gcsBlobBucket,
		},
		{
			name:       "Filesystem",
			blobBucket: fileBlobBucket,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CompressedWrite(ctx, test.blobBucket, objectName, originalData)
			if err != nil {
				t.Fatalf("we should be able to write: %s", err.Error())
			}

			objectReader, err := test.blobBucket.NewReader(ctx, objectName, nil)
			if err != nil {
				t.Fatalf("we should be able to read the object: %s", err.Error())
			}
			defer func() {
				err := objectReader.Close()
				if err != nil {
					t.Logf("closing the filereader: %s", err.Error())
				}
			}()

			r := lz4.NewReader(objectReader)
			uncompressedData, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("we should be able to uncompress the data: %v", err)
			}
			b, err := json.Marshal(originalData)
			if err != nil {
				t.Fatalf("we should be able to marshal this: %v", err)
			}
			if !bytes.Equal(b, bytes.TrimSpace(uncompressedData)) {
				t.Fatal("data should be identical")
			}
		})
	}
}

func TestDownloadProfile(t *testing.T) {
	ctx := context.Background()
	objectName := uuid.New().String()
	originalData := []byte(`{"samples":[1,2,3,4],"frames":[1,2,3,4]}`)

	var compressedData bytes.Buffer
	w := lz4.NewWriter(&compressedData)
	_, _ = w.Write(originalData)
	err := w.Close()
	if err != nil {
		t.Fatalf("we should be able to close the writer: %v", err)
	}

	tests := []struct {
		name       string
		blobBucket *blob.Bucket
	}{
		{
			name:       "GCS",
			blobBucket: gcsBlobBucket,
		},
		{
			name:       "Filesystem",
			blobBucket: fileBlobBucket,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wr, err := fileBlobBucket.NewWriter(ctx, objectName, nil)
			if err != nil {
				t.Fatalf("we should write an object: %s", err.Error())
			}

			_, err = wr.Write(compressedData.Bytes())
			if err != nil {
				t.Fatalf("we should write an object: %s", err.Error())
			}

			err = wr.Close()
			if err != nil {
				t.Fatalf("closing the filewriter: %s", err.Error())
			}

			var profile Profile
			err = UnmarshalCompressed(ctx, fileBlobBucket, objectName, &profile)
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
		})
	}
}

func TestDownloadProfileNotFound(t *testing.T) {
	ctx := context.Background()
	objectName := uuid.NewString()

	tests := []struct {
		name       string
		blobBucket *blob.Bucket
	}{
		{
			name:       "GCS",
			blobBucket: gcsBlobBucket,
		},
		{
			name:       "Filesystem",
			blobBucket: fileBlobBucket,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var profile Profile
			err := UnmarshalCompressed(ctx, test.blobBucket, objectName, &profile)
			if err == nil {
				t.Error("expecting an error, got nil")
			}

			if !errors.Is(err, ErrObjectNotFound) {
				t.Errorf("expecting an error of ErrObjectNotFound, instead got %s", err.Error())
			}
		})
	}
}

func BenchmarkGoJSON(b *testing.B) {
	b.ReportAllocs()
	testProfile, err := os.ReadFile("../../test/data/node.json")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		var result sample.Profile
		if err := gojson.Unmarshal(testProfile, &result); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJsonIterator(b *testing.B) {
	b.ReportAllocs()
	testProfile, err := os.ReadFile("../../test/data/node.json")
	if err != nil {
		b.Fatal(err)
	}
	for n := 0; n < b.N; n++ {
		var result sample.Profile
		if err := jsoniter.Unmarshal(testProfile, &result); err != nil {
			b.Fatal(err)
		}
	}
}
