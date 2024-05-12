package chunk

import (
	"encoding/json"
	"fmt"

	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
)

type (
	// Chunk is an implementation of the Sample V2 format.
	Chunk struct {
		ID         string `json:"chunk_id"`
		ProfilerID string `json:"profiler_id"`

		DebugMeta debugmeta.DebugMeta `json:"debug_meta"`

		Environment string            `json:"environment"`
		Platform    platform.Platform `json:"platform"`
		Release     string            `json:"release"`

		Version string `json:"version"`

		Profile Data `json:"profile"`

		OrganizationID uint64  `json:"organization_id"`
		ProjectID      uint64  `json:"project_id"`
		Received       float64 `json:"received"`
		RetentionDays  int     `json:"retention_days"`

		Measurements json.RawMessage
	}

	Data struct {
		Frames         []frame.Frame
		Samples        []Sample
		Stacks         [][]int
		ThreadMetadata map[string]map[string]string `json:"thread_metadata"`
	}

	Sample struct {
		StackID   int    `json:"stack_id"`
		ThreadID  string `json:"thread_id"`
		Timestamp float64
	}
)

func (c *Chunk) StoragePath() string {
	return fmt.Sprintf(
		"%d/%d/%s/%s",
		c.OrganizationID,
		c.ProjectID,
		c.ProfilerID,
		c.ID,
	)
}

func (c *Chunk) StartEndTimestamps() (float64, float64) {
	count := len(c.Profile.Samples)
	if count == 0 {
		return 0, 0
	}
	return c.Profile.Samples[0].Timestamp, c.Profile.Samples[count-1].Timestamp
}
