package profile

import "github.com/getsentry/vroom/internal/nodetree"

type (
	Trace interface {
		CallTrees() map[uint64][]*nodetree.Node
	}
)
