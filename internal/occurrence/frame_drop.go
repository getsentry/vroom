package occurrence

import (
	"math"
	"time"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
)

type (
	nodeStack struct {
		depth int
		n     *nodetree.Node
		st    []*nodetree.Node
	}

	frozenFrameStats struct {
		durationNS    uint64
		endNS         uint64
		minDurationNS uint64
		startLimitNS  uint64
		startNS       uint64
	}
)

func newFrozenFrameStats(endNS uint64, durationNS float64) frozenFrameStats {
	margin := uint64(math.Max(durationNS*marginPercent, float64(10*time.Millisecond)))
	s := frozenFrameStats{
		endNS:         endNS + margin,
		durationNS:    uint64(durationNS),
		minDurationNS: uint64(durationNS * minFrameDurationPercent),
	}
	if endNS >= (s.durationNS + margin) {
		s.startNS = endNS - s.durationNS - margin
	}
	s.startLimitNS = s.startNS + uint64(durationNS*0.20)
	return s
}

// nodeStackIfValid returns the nodeStack if we consider it valid as
// a frame drop cause.
func (s *frozenFrameStats) IsNodeStackValid(ns *nodeStack) bool {
	return s.startNS <= ns.n.StartNS &&
		ns.n.EndNS <= s.endNS &&
		s.minDurationNS <= s.durationNS &&
		ns.n.StartNS <= s.startLimitNS &&
		ns.n.IsApplication
}

const (
	FrameDrop Category = "frame_drop"

	marginPercent           float64 = 0.05
	minFrameDurationPercent float64 = 0.5
	startLimitPercent       float64 = 0.2
)

func findFrameDropCause(
	p profile.Profile,
	callTreesPerThreadID map[uint64][]*nodetree.Node,
	occurrences *[]*Occurrence,
) {
	frameDrops, exists := p.Measurements()["frozen_frame_renders"]
	if !exists {
		return
	}
	callTrees, exists := callTreesPerThreadID[p.Transaction().ActiveThreadID]
	if !exists {
		return
	}
	for _, mv := range frameDrops.Values {
		stats := newFrozenFrameStats(mv.ElapsedSinceStartNs, mv.Value)
		for _, root := range callTrees {
			st := make([]*nodetree.Node, 0, 128)
			cause := findFrameDropCauseFrame(
				root,
				stats,
				&st,
				0,
			)
			if cause == nil {
				continue
			}
			// We found a potential stacktrace responsible for this frozen frame
			stackTrace := make([]frame.Frame, 0, len(cause.st))
			for _, f := range cause.st {
				stackTrace = append(stackTrace, f.ToFrame())
			}
			*occurrences = append(
				*occurrences,
				NewOccurrence(p, nodeInfo{
					Category:   FrameDrop,
					Node:       *cause.n,
					StackTrace: stackTrace,
				}),
			)
			break
		}
	}
}

func findFrameDropCauseFrame(
	n *nodetree.Node,
	stats frozenFrameStats,
	st *[]*nodetree.Node,
	depth int,
) *nodeStack {
	*st = append(*st, n)
	defer func() {
		*st = (*st)[:len(*st)-1]
	}()
	var longest *nodeStack

	// Explore each branch to find the deepest valid node.
	for _, c := range n.Children {
		cause := findFrameDropCauseFrame(
			c,
			stats,
			st,
			depth+1,
		)
		if cause == nil {
			continue
		}
		if longest == nil {
			longest = cause
			continue
		}

		// Only keep the longest node.
		if cause.n.DurationNS > longest.n.DurationNS ||
			cause.n.DurationNS == longest.n.DurationNS && cause.depth > longest.depth {
			longest = cause
		}
	}

	var current *nodeStack

	// Create a nodeStack of the current node
	ns := &nodeStack{depth, n, nil}
	// Check if current node if valid.
	if stats.IsNodeStackValid(ns) {
		current = ns
	}

	if longest == nil && current == nil {
		return nil
	}

	// If we didn't find any valid node downstream, we return the current.
	if longest == nil {
		current.st = make([]*nodetree.Node, len(*st))
		copy(current.st, *st)
		return current
	}

	// If current is not valid or a node downstream is equal or longer, we return it.
	// We gave priority to the child instead of the current node.
	if current == nil || longest.n.DurationNS >= current.n.DurationNS {
		return longest
	}

	current.st = make([]*nodetree.Node, len(*st))
	copy(current.st, *st)
	return current
}
