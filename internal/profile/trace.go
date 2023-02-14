package profile

import (
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/speedscope"
)

type (
	Trace interface {
		CallTrees() map[uint64][]*nodetree.Node
		Speedscope() (speedscope.Output, error)
	}
)
