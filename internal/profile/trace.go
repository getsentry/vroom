package profile

import (
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/speedscope"
)

type (
	Trace interface {
		ActiveThreadID() uint64
		CallTrees() map[uint64][]*nodetree.Node
		DurationNS() uint64
		Speedscope() (speedscope.Output, error)
	}
)
