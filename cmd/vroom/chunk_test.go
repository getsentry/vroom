package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/getsentry/vroom/internal/chunk"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/storageutil"
	"github.com/getsentry/vroom/internal/testutil"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

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

func TestPostAndReadSampleChunk(t *testing.T) {
	profilerID := uuid.New().String()
	chunkID := uuid.New().String()
	chunkData := chunk.SampleChunk{
		ID:             chunkID,
		ProfilerID:     profilerID,
		Environment:    "dev",
		Platform:       "python",
		Release:        "1.2",
		OrganizationID: 1,
		ProjectID:      1,
		Version:        "2",
		Profile: chunk.SampleData{
			Frames: []frame.Frame{
				{
					Function: "test",
					InApp:    &testutil.True,
					Platform: platform.Python,
				},
			},
			Stacks: [][]int{
				{0},
			},
			Samples: []chunk.Sample{
				{StackID: 0, Timestamp: 1.0},
			},
		},
		Measurements: json.RawMessage("null"),
	}

	objectName := fmt.Sprintf(
		"%d/%d/%s/%s",
		chunkData.OrganizationID,
		chunkData.ProjectID,
		chunkData.ProfilerID,
		chunkData.ID,
	)

	tests := []struct {
		name       string
		blobBucket *blob.Bucket
		objectName string
	}{
		{
			name:       "Filesystem",
			blobBucket: fileBlobBucket,
			objectName: objectName,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := environment{
				storage:         test.blobBucket,
				profilingWriter: KafkaWriterMock{},
				config: ServiceConfig{
					ProfileChunksKafkaTopic: "snuba-profile-chunks",
				},
			}
			jsonValue, err := json.Marshal(chunkData)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest("POST", "/", bytes.NewBuffer(jsonValue))
			w := httptest.NewRecorder()

			// POST the chunk and check the we get a 204 response status code
			env.postChunk(w, req)
			resp := w.Result()
			defer resp.Body.Close()
			if resp.StatusCode != 204 {
				t.Fatalf("Expected status code 204. Found: %d", resp.StatusCode)
			}

			// read the chunk with UnmarshalCompressed and make sure that we can unmarshal
			// the data into the Chunk struct and that it matches the original
			var c chunk.Chunk
			err = storageutil.UnmarshalCompressed(
				context.Background(),
				test.blobBucket,
				objectName,
				&c,
			)
			if err != nil {
				t.Fatal(err)
			}
			if diff := testutil.Diff(&chunkData, c.Chunk()); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestPostAndReadAndroidChunk(t *testing.T) {
	profilerID := uuid.New().String()
	chunkID := uuid.New().String()
	chunkData := chunk.AndroidChunk{
		BuildID:    "1234",
		ID:         chunkID,
		ProfilerID: profilerID,
		DurationNS: 2_000_000,
		Platform:   "android",
		Timestamp:  1.123,
		Profile: profile.Android{
			Clock: "Dual",
			Events: []profile.AndroidEvent{
				{
					Action:   "Enter",
					ThreadID: 1,
					MethodID: 1,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Secs:  0,
								Nanos: 1000,
							},
						},
					},
				},
				{
					Action:   "Enter",
					ThreadID: 1,
					MethodID: 2,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Secs:  0,
								Nanos: 1500,
							},
						},
					},
				},
				{
					Action:   "Exit",
					ThreadID: 1,
					MethodID: 2,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Secs:  0,
								Nanos: 2000,
							},
						},
					},
				},
				{
					Action:   "Exit",
					ThreadID: 1,
					MethodID: 1,
					Time: profile.EventTime{
						Monotonic: profile.EventMonotonic{
							Wall: profile.Duration{
								Secs:  0,
								Nanos: 2500,
							},
						},
					},
				},
			},
			Methods: []profile.AndroidMethod{
				{
					ClassName: "class1",
					ID:        1,
					Name:      "method1",
					Signature: "()",
				},
				{
					ClassName: "class2",
					ID:        2,
					Name:      "method2",
					Signature: "()",
				},
			},
			StartTime: 0,
			Threads: []profile.AndroidThread{
				{
					ID:   1,
					Name: "main",
				},
			},
		}, // end profile
		OrganizationID: 1,
		ProjectID:      1,
		Received:       1.123,
		Measurements:   json.RawMessage("null"),
	}

	objectName := fmt.Sprintf(
		"%d/%d/%s/%s",
		chunkData.OrganizationID,
		chunkData.ProjectID,
		chunkData.ProfilerID,
		chunkData.ID,
	)

	tests := []struct {
		name       string
		blobBucket *blob.Bucket
		objectName string
	}{
		{
			name:       "Filesystem",
			blobBucket: fileBlobBucket,
			objectName: objectName,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := environment{
				storage:         test.blobBucket,
				profilingWriter: KafkaWriterMock{},
				config: ServiceConfig{
					ProfileChunksKafkaTopic: "snuba-profile-chunks",
				},
			}
			jsonValue, err := json.Marshal(chunkData)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest("POST", "/", bytes.NewBuffer(jsonValue))
			w := httptest.NewRecorder()

			// POST the chunk and check the we get a 204 response status code
			env.postChunk(w, req)
			resp := w.Result()
			defer resp.Body.Close()
			if resp.StatusCode != 204 {
				t.Fatalf("Expected status code 204. Found: %d", resp.StatusCode)
			}

			// read the chunk with UnmarshalCompressed and make sure that we can unmarshal
			// the data into the Chunk struct and that it matches the original
			var c chunk.Chunk
			err = storageutil.UnmarshalCompressed(
				context.Background(),
				test.blobBucket,
				objectName,
				&c,
			)
			if err != nil {
				t.Fatal(err)
			}
			if diff := testutil.Diff(&chunkData, c.Chunk()); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

type KafkaWriterMock struct{}

func (k KafkaWriterMock) WriteMessages(_ context.Context, _ ...kafka.Message) error {
	return nil
}

func (k KafkaWriterMock) Close() error {
	return nil
}
