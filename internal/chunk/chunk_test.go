package chunk

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalJSON(t *testing.T) {
	type chunkType int

	const (
		androidChunk chunkType = iota
		sampleChunk
	)

	tests := []struct {
		name        string
		input       string
		want        chunkType
		wantVersion string
	}{
		{
			name:        "absent version routes to AndroidChunk",
			input:       `{"chunk_id":"abc","profiler_id":"p1","platform":"android","profile":{"methods":[],"events":[]}}`,
			want:        androidChunk,
			wantVersion: "",
		},
		{
			name:        "empty version routes to AndroidChunk",
			input:       `{"chunk_id":"abc","profiler_id":"p1","version":"","platform":"android","profile":{"methods":[],"events":[]}}`,
			want:        androidChunk,
			wantVersion: "",
		},
		{
			name:        "2.android-trace version routes to AndroidChunk",
			input:       `{"chunk_id":"abc","profiler_id":"p1","version":"2.android-trace","platform":"android","profile":{"methods":[],"events":[]}}`,
			want:        androidChunk,
			wantVersion: "2.android-trace",
		},
		{
			name:        "android version 2 routes to SampleChunk",
			input:       `{"chunk_id":"abc","profiler_id":"p1","version":"2","platform":"android","profile":{"frames":[],"samples":[],"stacks":[]}}`,
			want:        sampleChunk,
			wantVersion: "2",
		},
		{
			name:        "non-android version 2 routes to SampleChunk",
			input:       `{"chunk_id":"abc","profiler_id":"p1","version":"2","platform":"python","profile":{"frames":[],"samples":[],"stacks":[]}}`,
			want:        sampleChunk,
			wantVersion: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Chunk
			err := json.Unmarshal([]byte(tt.input), &c)
			if err != nil {
				t.Fatalf("UnmarshalJSON returned error: %v", err)
			}
			switch tt.want {
			case androidChunk:
				ac, ok := c.chunk.(*AndroidChunk)
				if !ok {
					t.Fatalf("expected *AndroidChunk, got %T", c.chunk)
				}
				if ac.Version != tt.wantVersion {
					t.Errorf("expected Version %q, got %q", tt.wantVersion, ac.Version)
				}
			case sampleChunk:
				sc, ok := c.chunk.(*SampleChunk)
				if !ok {
					t.Fatalf("expected *SampleChunk, got %T", c.chunk)
				}
				if sc.Version != tt.wantVersion {
					t.Errorf("expected Version %q, got %q", tt.wantVersion, sc.Version)
				}
			}
			if id := c.GetID(); id != "abc" {
				t.Errorf("expected ID %q, got %q", "abc", id)
			}
			if profilerID := c.GetProfilerID(); profilerID != "p1" {
				t.Errorf("expected ProfilerID %q, got %q", "p1", profilerID)
			}
		})
	}
}

func TestUnmarshalJSONReturnsMalformedJSONError(t *testing.T) {
	var c Chunk
	if err := json.Unmarshal([]byte(`{"version":`), &c); err == nil {
		t.Fatal("expected UnmarshalJSON to return an error")
	}
}

func TestAndroidChunkPreservesVersionWhenMarshalled(t *testing.T) {
	input := `{"version":"2.android-trace","chunk_id":"abc","profiler_id":"p1","platform":"android","profile":{"methods":[],"events":[]}}`

	var c Chunk
	if err := json.Unmarshal([]byte(input), &c); err != nil {
		t.Fatalf("UnmarshalJSON returned error: %v", err)
	}

	ac, ok := c.chunk.(*AndroidChunk)
	if !ok {
		t.Fatalf("expected *AndroidChunk, got %T", c.chunk)
	}
	if ac.Version != "2.android-trace" {
		t.Errorf("expected Version %q, got %q", "2.android-trace", ac.Version)
	}

	out, err := json.Marshal(&c)
	if err != nil {
		t.Fatalf("MarshalJSON returned error: %v", err)
	}

	var roundTripped map[string]any
	if err := json.Unmarshal(out, &roundTripped); err != nil {
		t.Fatalf("failed to parse marshalled JSON: %v", err)
	}
	if v, ok := roundTripped["version"]; !ok || v != "2.android-trace" {
		t.Errorf("expected version %q in marshalled output, got %v", "2.android-trace", v)
	}
}
