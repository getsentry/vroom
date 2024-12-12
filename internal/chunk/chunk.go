package chunk

import (
	"encoding/json"
	"fmt"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/utils"
)

type (
	chunkInterface interface {
		GetEnvironment() string
		GetID() string
		GetOrganizationID() uint64
		GetPlatform() platform.Platform
		GetProfilerID() string
		GetProjectID() uint64
		GetReceived() float64
		GetRelease() string
		GetRetentionDays() int
		GetOptions() utils.Options
		GetFrameWithFingerprint(uint32) (frame.Frame, error)
		CallTrees(activeThreadID *string) (map[string][]*nodetree.Node, error)

		DurationMS() uint64
		EndTimestamp() float64
		SDKName() string
		SDKVersion() string
		StartTimestamp() float64
		StoragePath() string

		Normalize()
	}

	Chunk struct {
		chunk chunkInterface
	}
)

func New(c chunkInterface) Chunk {
	return Chunk{
		chunk: c,
	}
}

type version struct {
	Version string `json:"version"`
}

func (c *Chunk) UnmarshalJSON(b []byte) error {
	var v version
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	switch v.Version {
	case "":
		c.chunk = new(AndroidChunk)
	default:
		c.chunk = new(SampleChunk)
	}
	return json.Unmarshal(b, &c.chunk)
}

func (c Chunk) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.chunk)
}

func (c Chunk) Chunk() chunkInterface {
	return c.chunk
}

func StoragePath(OrganizationID uint64, ProjectID uint64, ProfilerID string, ID string) string {
	return fmt.Sprintf(
		"%d/%d/%s/%s",
		OrganizationID,
		ProjectID,
		ProfilerID,
		ID,
	)
}

func (c Chunk) GetEnvironment() string {
	return c.chunk.GetEnvironment()
}

func (c Chunk) GetID() string {
	return c.chunk.GetID()
}

func (c Chunk) GetOrganizationID() uint64 {
	return c.chunk.GetOrganizationID()
}

func (c Chunk) GetPlatform() platform.Platform {
	return c.chunk.GetPlatform()
}

func (c Chunk) GetProfilerID() string {
	return c.chunk.GetProfilerID()
}

func (c Chunk) GetProjectID() uint64 {
	return c.chunk.GetProjectID()
}

func (c Chunk) GetReceived() float64 {
	return c.chunk.GetReceived()
}

func (c Chunk) GetRelease() string {
	return c.chunk.GetRelease()
}

func (c Chunk) GetRetentionDays() int {
	return c.chunk.GetRetentionDays()
}

func (c Chunk) GetOptions() utils.Options {
	return c.chunk.GetOptions()
}

func (c Chunk) GetFrameWithFingerprint(f uint32) (frame.Frame, error) {
	return c.chunk.GetFrameWithFingerprint(f)
}

func (c Chunk) CallTrees(activeThreadID *string) (map[string][]*nodetree.Node, error) {
	return c.chunk.CallTrees(activeThreadID)
}

func (c Chunk) DurationMS() uint64 {
	return c.chunk.DurationMS()
}
func (c Chunk) EndTimestamp() float64 {
	return c.chunk.EndTimestamp()
}
func (c Chunk) SDKName() string {
	return c.chunk.SDKName()
}
func (c Chunk) SDKVersion() string {
	return c.chunk.SDKVersion()
}
func (c Chunk) StartTimestamp() float64 {
	return c.chunk.StartTimestamp()
}
func (c Chunk) StoragePath() string {
	return c.chunk.StoragePath()
}

func (c *Chunk) Normalize() {
	c.chunk.Normalize()
}
