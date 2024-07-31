package chunk

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
		ProfilerID     string
		ChunkID        string
		ThreadID       *string
		Start          uint64
		End            uint64
		Result         chan<- storageutil.ReadJobResult
	}

	ReadJobResult struct {
		Err      error
		Chunk    Chunk
		ThreadID *string
		Start    uint64
		End      uint64
	}
)

func (job ReadJob) Read() {
	var chunk Chunk

	err := storageutil.UnmarshalCompressed(
		job.Ctx,
		job.Storage,
		StoragePath(job.OrganizationID, job.ProjectID, job.ProfilerID, job.ChunkID),
		&chunk,
	)

	job.Result <- ReadJobResult{
		Err:      err,
		Chunk:    chunk,
		ThreadID: job.ThreadID,
		Start:    job.Start,
		End:      job.End,
	}
}

func (result ReadJobResult) Error() error {
	return result.Err
}
