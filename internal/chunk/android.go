package chunk

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/getsentry/vroom/internal/clientsdk"
	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/utils"
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

		Options utils.Options `json:"options,omitempty"`
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

func (c AndroidChunk) GetOptions() utils.Options {
	return c.Options
}

func (c AndroidChunk) GetFrameWithFingerprint(target uint32) (frame.Frame, error) {
	for _, m := range c.Profile.Methods {
		f := m.Frame()
		if f.Fingerprint() == target {
			return f, nil
		}
	}
	return frame.Frame{}, frame.ErrFrameNotFound
}

func (c *AndroidChunk) Normalize() {
}
