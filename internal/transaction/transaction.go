package transaction

type (
	Transaction struct {
		ActiveThreadID uint64
		DurationNS     uint64
		ID             string
		Name           string
		TraceID        string
	}
)
