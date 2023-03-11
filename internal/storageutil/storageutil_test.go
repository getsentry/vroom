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
	"github.com/dgraph-io/badger/v4"
	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/storageprovider"
	"github.com/google/uuid"
	"github.com/phayes/freeport"
	"github.com/pierrec/lz4/v4"

	gojson "github.com/goccy/go-json"
	jsoniter "github.com/json-iterator/go"
)

const bucketName = "profiles"

var gcsServer *fakestorage.Server
var badgerDB *badger.DB

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

	badgerDB, err = badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		log.Fatalf("couldn't create an in-memory badgerdb: %s", err.Error())
	}
	code := m.Run()

	err = badgerDB.Close()
	if err != nil {
		log.Printf("closing in-memory badgerdb: %s", err.Error())
	}

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

	t.Run("GCS", func(t *testing.T) {
		storageClient, err := storage.NewClient(ctx)
		if err != nil {
			t.Fatalf("we should be able to create a client: %v", err)
		}
		bucket := storageClient.Bucket(bucketName)
		err = CompressedWrite(ctx, &storageprovider.Gcs{BucketHandle: bucket}, objectName, originalData)
		if err != nil {
			t.Fatalf("we should be able to write: %v", err)
		}
		object, err := gcsServer.GetObject(bucketName, objectName)
		if err != nil {
			t.Fatalf("we should be able to read the object: %v", err)
		}
		r := lz4.NewReader(bytes.NewBuffer(object.Content))
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

	t.Run("Badger", func(t *testing.T) {
		err := CompressedWrite(ctx, &storageprovider.Badger{DB: badgerDB}, objectName, originalData)
		if err != nil {
			t.Fatalf("we should be able to write: %s", err.Error())
		}

		var valueReader io.Reader
		err = badgerDB.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(objectName))
			if err != nil {
				txn.Discard()
				return err
			}

			value, err := item.ValueCopy(nil)
			if err != nil {
				txn.Discard()
				return err
			}

			valueReader = bytes.NewReader(value)
			return txn.Commit()
		})
		if err != nil {
			t.Fatalf("we should be able to read the object: %s", err.Error())
		}

		r := lz4.NewReader(valueReader)
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

	t.Run("GCS", func(t *testing.T) {
		gcsServer.CreateObject(fakestorage.Object{
			ObjectAttrs: fakestorage.ObjectAttrs{
				BucketName: bucketName,
				Name:       objectName,
			},
			Content: compressedData.Bytes(),
		})

		storageClient, err := storage.NewClient(ctx)
		if err != nil {
			t.Fatalf("we should be able to create a client: %v", err)
		}
		bucket := storageClient.Bucket(bucketName)
		var profile Profile
		err = UnmarshalCompressed(ctx, &storageprovider.Gcs{BucketHandle: bucket}, objectName, &profile)
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

	t.Run("Badger", func(t *testing.T) {
		err := badgerDB.Update(func(txn *badger.Txn) error {
			err := txn.Set([]byte(objectName), compressedData.Bytes())
			if err != nil {
				txn.Discard()
				return err
			}

			return txn.Commit()
		})
		if err != nil {
			t.Fatalf("we should be write an object: %s", err.Error())
		}

		var profile Profile
		err = UnmarshalCompressed(ctx, &storageprovider.Badger{DB: badgerDB}, objectName, &profile)
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
