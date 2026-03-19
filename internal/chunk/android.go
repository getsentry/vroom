package chunk

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/getsentry/vroom/internal/clientsdk"
	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/options"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
)

type (
	AndroidChunk struct {
		BuildID    string `json:"build_id,omitempty"`
		ID         string `json:"chunk_id"`
		ProfilerID string `json:"profiler_id"`

		DebugMeta debugmeta.DebugMeta `json:"debug_meta"`

		ClientSDK   clientsdk.ClientSDK `json:"client_sdk"`
		DurationNS  uint64              `json:"duration_ns"`
		Environment string              `json:"environment"`
		Platform    platform.Platform   `json:"platform"`
		Release     string              `json:"release"`
		Timestamp   float64             `json:"timestamp"`

		Profile      profile.Android `json:"profile"`
		Measurements json.RawMessage `json:"measurements"`

		OrganizationID uint64  `json:"organization_id"`
		ProjectID      uint64  `json:"project_id"`
		Received       float64 `json:"received"`
		RetentionDays  int     `json:"retention_days"`

		Options options.Options `json:"options,omitempty"`
	}
)

func (c AndroidChunk) StoragePath() string {
	return StoragePath(
		c.OrganizationID,
		c.ProjectID,
		c.ProfilerID,
		c.ID,
	)
}

func (c AndroidChunk) DurationMS() uint64 {
	return uint64(time.Duration(c.DurationNS).Milliseconds())
}

func (c AndroidChunk) CallTrees(_ *string) (map[string][]*nodetree.Node, error) {
	c.Profile.SdkStartTime = uint64(c.StartTimestamp() * 1e9)
	callTrees := c.Profile.CallTrees()
	stringThreadCallTrees := make(map[string][]*nodetree.Node)
	for tid, callTree := range callTrees {
		threadID := strconv.FormatUint(tid, 10)
		stringThreadCallTrees[threadID] = callTree
	}
	return stringThreadCallTrees, nil
}

func (c AndroidChunk) SDKName() string {
	return c.ClientSDK.Name
}

func (c AndroidChunk) SDKVersion() string {
	return c.ClientSDK.Version
}

func (c AndroidChunk) EndTimestamp() float64 {
	return c.Timestamp + float64(c.DurationNS)*1e-9
}

func (c AndroidChunk) GetEnvironment() string {
	return c.Environment
}

func (c AndroidChunk) GetID() string {
	return c.ID
}

func (c AndroidChunk) GetPlatform() platform.Platform {
	return c.Platform
}

func (c AndroidChunk) GetProfilerID() string {
	return c.ProfilerID
}

func (c AndroidChunk) GetProjectID() uint64 {
	return c.ProjectID
}

func (c AndroidChunk) GetReceived() float64 {
	return c.Received
}

func (c AndroidChunk) GetRelease() string {
	return c.Release
}

func (c AndroidChunk) GetRetentionDays() int {
	return c.RetentionDays
}

func (c AndroidChunk) StartTimestamp() float64 {
	return c.Timestamp
}

func (c AndroidChunk) GetOrganizationID() uint64 {
	return c.OrganizationID
}

func (c AndroidChunk) GetOptions() options.Options {
	return c.Options
}

func (c AndroidChunk) GetFrameWithFingerprint(target uint32) (frame.Frame, error) {
	// Try exact match first
	for _, m := range c.Profile.Methods {
		f := m.Frame()
		if f.Fingerprint() == target {
			return f, nil
		}
	}
	
	// Build frames array for fallback matching
	frames := make([]frame.Frame, 0, len(c.Profile.Methods))
	for _, m := range c.Profile.Methods {
		frames = append(frames, m.Frame())
	}
	
	// Try fallback with fingerprint variations
	matchedFrame, usedFallback, err := frame.FindFrameByFingerprintWithFallback(frames, target)
	if err == nil && usedFallback {
		slog.Warn(
			"Frame matched using fallback fingerprint computation",
			"target_fingerprint", target,
			"matched_frame_fingerprint", matchedFrame.Fingerprint(),
			"matched_function", matchedFrame.Function,
			"matched_module", matchedFrame.ModuleOrPackage(),
			"chunk_id", c.ID,
			"profiler_id", c.ProfilerID,
		)
		return matchedFrame, nil
	}
	
	return frame.Frame{}, frame.ErrFrameNotFound
}

func (c *AndroidChunk) Normalize() {
}
