package chunk

import (
	"context"

	"github.com/getsentry/vroom/internal/nodetree"
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
		TransactionID  string
		ThreadID       *string
		Start          uint64
		End            uint64
		Result         chan<- storageutil.ReadJobResult
	}

	ReadJobResult struct {
		Err           error
		Chunk         *Chunk
		TransactionID string
		ThreadID      *string
		Start         uint64
		End           uint64
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
		Chunk:         &chunk,
		TransactionID: job.TransactionID,
		ThreadID:      job.ThreadID,
		Start:         job.Start,
		End:           job.End,
	}
}

func (result ReadJobResult) Error() error {
	return result.Err
}

type (
	CallTreesReadJob ReadJob

	CallTreesReadJobResult struct {
		Err           error
		CallTrees     map[string][]*nodetree.Node
		Chunk         *Chunk
		TransactionID string
		ThreadID      *string
		Start         uint64
		End           uint64
	}
)

func (job CallTreesReadJob) Read() {
	var chunk Chunk

	err := storageutil.UnmarshalCompressed(
		job.Ctx,
		job.Storage,
		StoragePath(job.OrganizationID, job.ProjectID, job.ProfilerID, job.ChunkID),
		&chunk,
	)
	if err != nil {
		job.Result <- CallTreesReadJobResult{Err: err}
		return
	}

	callTrees, err := chunk.CallTrees(job.ThreadID)

	job.Result <- CallTreesReadJobResult{
		Err:           err,
		CallTrees:     callTrees,
		Chunk:         &chunk,
		TransactionID: job.TransactionID,
		ThreadID:      job.ThreadID,
		Start:         job.Start,
		End:           job.End,
	}
}

func (result CallTreesReadJobResult) Error() error {
	return result.Err
}
