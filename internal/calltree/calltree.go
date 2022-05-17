package calltree

// NoEndTime signifies that a function call or call tree does not have an end
// time because the data necessary to compute the end time was missing.
const NoEndTime uint64 = 0

type CallTreeP struct {
	Address     string
	ThreadID    uint64
	StartTimeNs uint64
	EndTimeNs   uint64
	SelfTimeNs  uint64
	ProfileID   string
	SessionKey  string
	ThreadName  string
	Children    []*CallTreeP
}
