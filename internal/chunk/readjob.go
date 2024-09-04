package chunk

import (
	"context"

	"github.com/getsentry/vroom/internal/storageutil"
	"github.com/getsentry/vroom/internal/utils"
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
		TransactionID  string
		ThreadID       *string
		Start          uint64
		End            uint64
		Intervals      map[string][]utils.Interval
		Result         chan<- storageutil.ReadJobResult
	}

	ReadJobResult struct {
		Err           error
		Chunk         Chunk
		TransactionID string
		ThreadID      *string
		Start         uint64
		End           uint64
		Intervals     map[string][]utils.Interval
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
		Err:           err,
		Chunk:         chunk,
		TransactionID: job.TransactionID,
		ThreadID:      job.ThreadID,
		Start:         job.Start,
		End:           job.End,
		Intervals:     job.Intervals,
	}
}

func (result ReadJobResult) Error() error {
	return result.Err
}
