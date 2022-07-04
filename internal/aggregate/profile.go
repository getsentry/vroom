package aggregate

import "github.com/getsentry/vroom/internal/nodetree"

type Profile interface {
	CallTrees() map[uint64][]*nodetree.Node
}
