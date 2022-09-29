package profile

import (
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/speedscope"
)

type Typescript struct{}

func (t Typescript) CallTrees() map[uint64][]*nodetree.Node {
	return make(map[uint64][]*nodetree.Node)
}

func (t Typescript) Speedscope() (speedscope.Output, error) {
	return speedscope.Output{}, nil
}
