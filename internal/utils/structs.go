package utils

type (
	TransactionProfileCandidate struct {
		ProjectID uint64 `json:"project_id"`
		ProfileID string `json:"profile_id"`
	}

	ContinuousProfileCandidate struct {
		ProjectID  uint64  `json:"project_id"`
		ProfilerID string  `json:"profiler_id"`
		ChunkID    string  `json:"chunk_id"`
		ThreadID   *string `json:"thread_id"`
		Start      uint64  `json:"start,string"`
		End        uint64  `json:"end,string"`
	}
)
