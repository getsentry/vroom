package chunk

import (
	"context"

	"github.com/getsentry/vroom/internal/nodetree"
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

type (
	CallTreesReadJob ReadJob

	CallTreesReadJobResult struct {
		Err           error
		CallTrees     map[string][]*nodetree.Node
		Chunk         Chunk
		TransactionID string
		ThreadID      *string
		Start         uint64
		End           uint64
		Intervals     map[string][]utils.Interval
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

	callTrees := make(map[string][]*nodetree.Node)
	for tid := range job.Intervals {
		if tid == "" {
			callTrees, err = chunk.CallTrees(nil)
			if err != nil {
				job.Result <- CallTreesReadJobResult{Err: err}
				return
			}
			// we've already computed the callTrees for all the
			// tids that we might need so we can bail out
			break
		}
		if _, ok := callTrees[tid]; ok {
			// if a former interval already had the same
			// tid we've already computed that callTree
			// and we can bail out early
			continue
		}
		callTree, err := chunk.CallTrees(&tid)
		if err != nil {
			job.Result <- CallTreesReadJobResult{Err: err}
			return
		}
		callTrees[tid] = callTree[tid]
	}
	job.Result <- CallTreesReadJobResult{
		Err:           err,
		CallTrees:     callTrees,
		Chunk:         chunk,
		TransactionID: job.TransactionID,
		ThreadID:      job.ThreadID,
		Start:         job.Start,
		End:           job.End,
	}
}

func (result CallTreesReadJobResult) Error() error {
	return result.Err
}
