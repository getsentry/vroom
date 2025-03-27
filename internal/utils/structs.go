package utils

type (
	Interval struct {
		Start          uint64 `json:"start,string"`
		End            uint64 `json:"end,string"`
		ActiveThreadID string `json:"active_thread_id,omitempty"`
	}

	TransactionProfileCandidate struct {
		ProjectID uint64 `json:"project_id"`
		ProfileID string `json:"profile_id"`
	}

	ContinuousProfileCandidate struct {
		ProjectID     uint64  `json:"project_id"`
		ProfilerID    string  `json:"profiler_id"`
		ChunkID       string  `json:"chunk_id"`
		TransactionID string  `json:"transaction_id"`
		ThreadID      *string `json:"thread_id"`
		Start         uint64  `json:"start,string"`
		End           uint64  `json:"end,string"`
	}

	// ExampleMetadata and FunctionMetrics have been moved here, although they'd
	// belong more to the metrics package, in order to avoid the circular dependency
	// hell that'd be introduced following the optimization to support metrics
	// generation within the flamegraph logic.
	ExampleMetadata struct {
		ProjectID     uint64  `json:"project_id,omitempty"`
		ProfileID     string  `json:"profile_id,omitempty"`
		ProfilerID    string  `json:"profiler_id,omitempty"`
		ChunkID       string  `json:"chunk_id,omitempty"`
		TransactionID string  `json:"transaction_id,omitempty"`
		ThreadID      *string `json:"thread_id,omitempty"`
		Start         float64 `json:"start,omitempty"`
		End           float64 `json:"end,omitempty"`
	}

	FunctionMetrics struct {
		Name        string            `json:"name"`
		Package     string            `json:"package"`
		Fingerprint uint64            `json:"fingerprint"`
		InApp       bool              `json:"in_app"`
		P75         uint64            `json:"p75"`
		P95         uint64            `json:"p95"`
		P99         uint64            `json:"p99"`
		Avg         float64           `json:"avg"`
		Sum         uint64            `json:"sum"`
		Count       uint64            `json:"count"`
		Worst       ExampleMetadata   `json:"worst"`
		Examples    []ExampleMetadata `json:"examples"`
	}
)

func NewExampleFromProfileID(
	projectID uint64,
	profileID string,
	start uint64,
	end uint64,
) ExampleMetadata {
	return ExampleMetadata{
		ProjectID: projectID,
		ProfileID: profileID,
		Start:     float64(start) / 1e9,
		End:       float64(end) / 1e9,
	}
}

func NewExampleFromProfilerChunk(
	projectID uint64,
	profilerID string,
	chunkID string,
	transactionID string,
	threadID *string,
	start uint64,
	end uint64,
) ExampleMetadata {
	return ExampleMetadata{
		ProjectID:     projectID,
		ProfilerID:    profilerID,
		ChunkID:       chunkID,
		TransactionID: transactionID,
		ThreadID:      threadID,
		Start:         float64(start) / 1e9,
		End:           float64(end) / 1e9,
	}
}
