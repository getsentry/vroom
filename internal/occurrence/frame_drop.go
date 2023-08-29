package occurrence

import (
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
)

const (
	FrameDrop Category = "frame_drop"
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
		for _, root := range callTrees {
			st := make([]*nodetree.Node, 0, 128)
			cause := findFrameDropCauseFrame(
				root,
				mv.ElapsedSinceStartNs-uint64(mv.Value),
				mv.ElapsedSinceStartNs,
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
	frozenFrameStartNS, frozenFrameEndNS uint64,
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
			frozenFrameStartNS,
			frozenFrameEndNS,
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

	// Create a nodeStack of the current node
	ns := &nodeStack{depth, n, nil}
	// Check if current node if valid.
	current := nodeStackIfValid(
		ns,
		frozenFrameStartNS,
		frozenFrameEndNS,
	)

	// If we didn't find any valid node downstream, we return the current.
	if longest == nil {
		ns.st = make([]*nodetree.Node, len(*st))
		copy(ns.st, *st)
		return current
	}

	// If current is not valid or a node downstream is equal or longer, we return it.
	// We gave priority to the child instead of the current node.
	if current == nil || longest.n.DurationNS >= current.n.DurationNS {
		return longest
	}

	ns.st = make([]*nodetree.Node, len(*st))
	copy(ns.st, *st)
	return current
}

// nodeStackIfValid returns the nodeStack if we consider it valid as
// a frame drop cause.
func nodeStackIfValid(
	ns *nodeStack,
	frozenFrameStartNS, frozenFrameEndNS uint64,
) *nodeStack {
	if ns.n.StartNS >= frozenFrameStartNS &&
		ns.n.EndNS <= frozenFrameEndNS &&
		ns.n.IsApplication {
		return ns
	}
	return nil
}
