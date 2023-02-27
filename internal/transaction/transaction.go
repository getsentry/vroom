package transaction

type (
	Transaction struct {
		ActiveThreadID uint64 `json:"active_thread_id"`
		DurationNS     uint64 `json:"duration_ns,omitempty"`
		ID             string `json:"id"`
		Name           string `json:"name"`
		TraceID        string `json:"trace_id"`
	}
)
