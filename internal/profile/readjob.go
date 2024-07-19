package profile

import (
	"context"

	"github.com/getsentry/vroom/internal/storageutil"
	"gocloud.dev/blob"
)

type (
	ReadJob struct {
		Ctx            context.Context
		Storage        *blob.Bucket
		OrganizationID uint64
		ProjectID      uint64
		ProfileID      string
		Result         chan<- storageutil.ReadJobResult
	}

	ReadJobResult struct {
		Err     error
		Profile Profile
	}
)

func (job ReadJob) Read() {
	var profile Profile

	err := storageutil.UnmarshalCompressed(
		job.Ctx,
		job.Storage,
		StoragePath(job.OrganizationID, job.ProjectID, job.ProfileID),
		&profile,
	)

	job.Result <- ReadJobResult{Profile: profile, Err: err}
}

func (result ReadJobResult) Error() error {
	return result.Err
}
