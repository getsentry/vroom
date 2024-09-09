package utils

type (
	Interval struct {
		Start          uint64  `json:"start,string"`
		End            uint64  `json:"end,string"`
		ActiveThreadID *string `json:"active_thread_id,omitempty"`
	}

	TransactionProfileCandidate struct {
		ProjectID uint64 `json:"project_id"`
		ProfileID string `json:"profile_id"`
	}

	ContinuousProfileCandidate struct {
		ProjectID     uint64                `json:"project_id"`
		ProfilerID    string                `json:"profiler_id"`
		ChunkID       string                `json:"chunk_id"`
		TransactionID string                `json:"transaction_id"`
		ThreadID      *string               `json:"thread_id"`
		Start         uint64                `json:"start,string"`
		End           uint64                `json:"end,string"`
		Intervals     map[string][]Interval `json:"-"`
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
) ExampleMetadata {
	return ExampleMetadata{
		ProjectID: projectID,
		ProfileID: profileID,
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

func MergeContinuousProfileCandidate(candidates []ContinuousProfileCandidate) []ContinuousProfileCandidate {
	chunkToIdx := map[string]int{}
	newCandidates := []ContinuousProfileCandidate{}
	for _, c := range candidates {
		newInterval := Interval{
			Start: c.Start,
			End:   c.End,
		}
		tid := ""
		if c.ThreadID != nil {
			tid = *c.ThreadID
		}
		// if we already have a candidate with such chunkID, add the interval
		if idx, ok := chunkToIdx[c.ChunkID]; ok {
			// if there is already an interval for a given thread ID
			// add the interval to that list
			if _, ok := newCandidates[idx].Intervals[tid]; ok {
				intervals := newCandidates[idx].Intervals[tid]
				intervals = append(intervals, newInterval)
				if c.Start != 0 && c.End != 0 {
					newCandidates[idx].Intervals[tid] = intervals
				}
			} else {
				// else add a new list of intervals for such threadID
				if c.Start != 0 && c.End != 0 {
					newCandidates[idx].Intervals[tid] = []Interval{newInterval}
				}
			}
		} else {
			// else add a new candidate
			chunkToIdx[c.ChunkID] = len(newCandidates)
			candidate := ContinuousProfileCandidate{
				ProjectID:     c.ProjectID,
				ProfilerID:    c.ProfilerID,
				ChunkID:       c.ChunkID,
				TransactionID: c.TransactionID,
				Intervals:     map[string][]Interval{},
			}
			if c.Start != 0 && c.End != 0 {
				candidate.Intervals[tid] = []Interval{newInterval}
			}
			newCandidates = append(newCandidates, candidate)
		}
	} // end loop candidates
	return newCandidates
}
