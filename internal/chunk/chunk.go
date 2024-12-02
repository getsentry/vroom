package chunk

import (
	"fmt"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/utils"
)

type (
	Chunk interface {
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

		DurationMS() uint64
		EndTimestamp() float64
		SDKName() string
		SDKVersion() string
		StartTimestamp() float64
		StoragePath() string

		Normalize()
	}
)

func StoragePath(OrganizationID uint64, ProjectID uint64, ProfilerID string, ID string) string {
	return fmt.Sprintf(
		"%d/%d/%s/%s",
		OrganizationID,
		ProjectID,
		ProfilerID,
		ID,
	)
}
