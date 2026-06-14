package chunk

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/testutil"
)

func TestAttachments(t *testing.T) {
	payload := `{
		"version": "2",
		"chunk_id": "0432a0a4c25f4697bf9f0a2fcbe6a814",
		"attachments": [
			{
				"name": "raw_profile",
				"content_type": "application/x-perfetto",
				"stored_id": "aef123345"
			}
		]
	}`

	var c Chunk
	if err := json.Unmarshal([]byte(payload), &c); err != nil {
		t.Fatal(err)
	}
	sc, ok := c.Chunk().(*SampleChunk)
	if !ok {
		t.Fatalf("expected *SampleChunk, got %T", c.Chunk())
	}
	want := []Attachment{
		{Name: "raw_profile", ContentType: "application/x-perfetto", StoredID: "aef123345"},
	}
	if diff := testutil.Diff(sc.Attachments, want); diff != "" {
		t.Fatalf("Result mismatch: got - want +\n%s", diff)
	}
	if diff := testutil.Diff(c.GetAttachments(), want); diff != "" {
		t.Fatalf("Result mismatch: got - want +\n%s", diff)
	}

	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"attachments":[{"name":"raw_profile","content_type":"application/x-perfetto","stored_id":"aef123345"}]`)) {
		t.Errorf("expected serialized chunk to contain the attachments: %s", b)
	}
}

func TestAttachmentsOmittedWhenEmpty(t *testing.T) {
	payload := `{"version": "2", "chunk_id": "0432a0a4c25f4697bf9f0a2fcbe6a814"}`

	var c Chunk
	if err := json.Unmarshal([]byte(payload), &c); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte(`"attachments"`)) {
		t.Errorf("expected attachments to be omitted: %s", b)
	}
}

// A chunk with platform=android but a version set uses the sample v2 format
// and must not be treated as a legacy android chunk.
func TestUnmarshalAndroidPlatformWithVersionAsSampleChunk(t *testing.T) {
	payload := `{
		"version": "2",
		"platform": "android",
		"chunk_id": "0432a0a4c25f4697bf9f0a2fcbe6a814"
	}`

	var c Chunk
	if err := json.Unmarshal([]byte(payload), &c); err != nil {
		t.Fatal(err)
	}
	sc, ok := c.Chunk().(*SampleChunk)
	if !ok {
		t.Fatalf("expected *SampleChunk, got %T", c.Chunk())
	}
	if sc.Platform != platform.Android {
		t.Errorf("expected platform %q, got %q", platform.Android, sc.Platform)
	}
}

// Attachments are only supported for sample chunks:
// the field is dropped on android chunks.
func TestAttachmentsIgnoredOnAndroidChunks(t *testing.T) {
	payload := `{
		"chunk_id": "0432a0a4c25f4697bf9f0a2fcbe6a814",
		"attachments": [
			{
				"name": "raw_profile",
				"content_type": "application/x-perfetto",
				"stored_id": "aef123345"
			}
		]
	}`

	var c Chunk
	if err := json.Unmarshal([]byte(payload), &c); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Chunk().(*AndroidChunk); !ok {
		t.Fatalf("expected *AndroidChunk, got %T", c.Chunk())
	}
	if got := c.GetAttachments(); len(got) != 0 {
		t.Errorf("expected GetAttachments to return an empty list, got %v", got)
	}

	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte(`"attachments"`)) {
		t.Errorf("expected attachments to be omitted: %s", b)
	}
}
