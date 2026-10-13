package occurrence

import (
	"context"
	"errors"
	"testing"

	"github.com/getsentry/vroom/internal/chunk"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/storageutil"
	"github.com/getsentry/vroom/internal/testutil"
)

type regressionFrameLookup interface {
	GetFrameWithFingerprint(uint32) (frame.Frame, error)
	GetFrameForRegression(uint32) (frame.Frame, error)
}

type regressionFrameSource struct {
	name     string
	lookup   regressionFrameLookup
	result   storageutil.ReadJobResult
	platform platform.Platform
	frames   []frame.Frame
}

func regressionFrameSources(names ...string) []regressionFrameSource {
	frames := make([]frame.Frame, len(names))
	methods := make([]profile.AndroidMethod, len(names))
	androidFrames := make([]frame.Frame, len(names))
	for i, name := range names {
		frames[i] = frame.Frame{Function: name}
		methods[i] = profile.AndroidMethod{ID: uint64(i + 1), Name: name}
		androidFrames[i] = methods[i].Frame()
	}

	sampled := profile.New(&sample.Profile{RawProfile: sample.RawProfile{
		Platform: platform.Node,
		Trace:    sample.Trace{Frames: frames},
	}})
	android := profile.New(&profile.LegacyProfile{
		RawProfile: profile.RawProfile{Platform: platform.Android},
		Trace:      &profile.Android{Methods: methods},
	})
	sampledChunk := chunk.New(&chunk.SampleChunk{
		Platform: platform.Node,
		Profile:  chunk.SampleData{Frames: frames},
	})
	androidChunk := chunk.New(&chunk.AndroidChunk{
		Platform: platform.Android,
		Profile:  profile.Android{Methods: methods},
	})

	return []regressionFrameSource{
		{
			name:     "sample profile",
			lookup:   &sampled,
			result:   profile.ReadJobResult{Profile: &sampled},
			platform: platform.Node,
			frames:   frames,
		},
		{
			name:     "android profile",
			lookup:   &android,
			result:   profile.ReadJobResult{Profile: &android},
			platform: platform.Android,
			frames:   androidFrames,
		},
		{
			name:     "sample chunk",
			lookup:   sampledChunk,
			result:   chunk.ReadJobResult{Chunk: &sampledChunk},
			platform: platform.Node,
			frames:   frames,
		},
		{
			name:     "android chunk",
			lookup:   androidChunk,
			result:   chunk.ReadJobResult{Chunk: &androidChunk},
			platform: platform.Android,
			frames:   androidFrames,
		},
	}
}

func TestGetFrameForRegression(t *testing.T) {
	tests := []struct {
		name        string
		fingerprint uint32
		wantErr     error
	}{
		{name: "legacy fingerprint", fingerprint: 2431592238, wantErr: frame.ErrFrameNotFound},
		{name: "vroomrs fingerprint", fingerprint: 1176630088},
		{name: "unknown fingerprint", fingerprint: 0, wantErr: frame.ErrFrameNotFound},
	}

	for _, source := range regressionFrameSources("work") {
		t.Run(source.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					got, err := source.lookup.GetFrameForRegression(tt.fingerprint)
					if !errors.Is(err, tt.wantErr) {
						t.Fatalf("GetFrameForRegression() error = %v, want %v", err, tt.wantErr)
					}
					want := frame.Frame{}
					if tt.wantErr == nil {
						want = source.frames[0]
					}
					if diff := testutil.Diff(want, got); diff != "" {
						t.Fatalf("GetFrameForRegression() mismatch (-want +got):\n%s", diff)
					}

					gotPlatform, gotFrame, err := getPlatformAndFrame(context.Background(), source.result, tt.fingerprint)
					if !errors.Is(err, tt.wantErr) {
						t.Fatalf("getPlatformAndFrame() error = %v, want %v", err, tt.wantErr)
					}
					if gotPlatform != source.platform {
						t.Fatalf("getPlatformAndFrame() platform = %s, want %s", gotPlatform, source.platform)
					}
					if diff := testutil.Diff(want, gotFrame); diff != "" {
						t.Fatalf("getPlatformAndFrame() mismatch (-want +got):\n%s", diff)
					}
				})
			}

			got, err := source.lookup.GetFrameWithFingerprint(2431592238)
			if err != nil {
				t.Fatalf("legacy lookup failed: %v", err)
			}
			if diff := testutil.Diff(source.frames[0], got); diff != "" {
				t.Fatalf("legacy lookup mismatch (-want +got):\n%s", diff)
			}
			got, err = source.lookup.GetFrameWithFingerprint(1176630088)
			if !errors.Is(err, frame.ErrFrameNotFound) {
				t.Fatalf("legacy lookup accepted vroomrs fingerprint: %v", err)
			}
			if diff := testutil.Diff(frame.Frame{}, got); diff != "" {
				t.Fatalf("legacy lookup returned a frame (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetFrameForRegressionEmpty(t *testing.T) {
	for _, source := range regressionFrameSources() {
		t.Run(source.name, func(t *testing.T) {
			got, err := source.lookup.GetFrameForRegression(1176630088)
			if !errors.Is(err, frame.ErrFrameNotFound) {
				t.Fatalf("GetFrameForRegression() error = %v, want %v", err, frame.ErrFrameNotFound)
			}
			if diff := testutil.Diff(frame.Frame{}, got); diff != "" {
				t.Fatalf("GetFrameForRegression() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetPlatformAndFrameReadError(t *testing.T) {
	wantErr := errors.New("read failed")
	for _, result := range []storageutil.ReadJobResult{
		profile.ReadJobResult{Err: wantErr},
		chunk.ReadJobResult{Err: wantErr},
	} {
		_, _, err := getPlatformAndFrame(context.Background(), result, 1176630088)
		if !errors.Is(err, wantErr) {
			t.Fatalf("getPlatformAndFrame() error = %v, want %v", err, wantErr)
		}
	}
}
