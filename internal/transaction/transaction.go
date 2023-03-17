package transaction

import "github.com/getsentry/vroom/internal/types"

type (
	Transaction struct {
		ActiveThreadID types.Uint64 `json:"active_thread_id"`
		DurationNS     uint64       `json:"duration_ns,omitempty"`
		ID             string       `json:"id"`
		Name           string       `json:"name"`
		TraceID        string       `json:"trace_id"`
	}
)

func (t Transaction) GetActiveThreadID() uint64 {
	return uint64(t.ActiveThreadID)
}
