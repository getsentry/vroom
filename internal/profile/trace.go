package profile

import (
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/speedscope"
)

type (
	Trace interface {
		ActiveThreadID() uint64
		CallTrees() map[uint64][]*nodetree.Node
		Speedscope() (speedscope.Output, error)
		GetFrameWithFingerprint(uint32) (frame.Frame, error)
	}
)
