package occurrence

import (
	"sort"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
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
			frameDropInfo := findFrameDropFrame(
				root,
				mv.ElapsedSinceStartNs-uint64(mv.Value),
				mv.ElapsedSinceStartNs,
				&st,
			)
			// We found a potential stacktrace responsible for this frozen frame
			if frameDropInfo != nil {
				// Create occurrences.
				stackTrace := make([]frame.Frame, 0, len(frameDropInfo.StackTrace))
				for _, f := range frameDropInfo.StackTrace {
					stackTrace = append(stackTrace, f.ToFrame())
				}
				*occurrences = append(
					*occurrences,
					NewOccurrence(p, nodeInfo{Node: frameDropInfo.Node, StackTrace: stackTrace}),
				)
				break
			}
		}
	}
}

type (
	indexDuration struct {
		durationNS uint64
		index      int
	}

	frameDropInfo struct {
		Node       nodetree.Node
		StackTrace []*nodetree.Node
	}
)

func findFrameDropFrame(
	n *nodetree.Node,
	frozenFrameStartNS uint64,
	frozenFrameEndNS uint64,
	st *[]*nodetree.Node,
) *frameDropInfo {
	*st = append(*st, n)
	defer func() {
		*st = (*st)[:len(*st)-1]
	}()
	if len(n.Children) == 0 {
		stackTrace := *st
		inAppIndex := -1
		for i := len(stackTrace) - 1; i >= 0; i-- {
			f := stackTrace[i]
			if *f.Frame.InApp {
				inAppIndex = i
				break
			}
		}
		// We didn't find an in app frame in the stack, occurrence is discarded
		if inAppIndex == -1 {
			return nil
		}
		// We truncate to focus on the last in app frame
		stackTrace = stackTrace[:inAppIndex]
		ni := frameDropInfo{
			Node: *stackTrace[len(stackTrace)-1],
		}
		ni.Node.Children = nil
		ni.StackTrace = make([]*nodetree.Node, len(stackTrace))
		copy(ni.StackTrace, stackTrace)
		return &ni
	} else if len(n.Children) == 1 {
		return findFrameDropFrame(n.Children[0], frozenFrameStartNS, frozenFrameEndNS, st)
	}
	// Select candidates to explore next
	candidates := make([]indexDuration, 0, len(n.Children))
	for i, c := range n.Children {
		if c.EndNS < frozenFrameStartNS || c.StartNS > frozenFrameEndNS {
			continue
		}
		candidates = append(
			candidates,
			indexDuration{
				c.DurationNS,
				i,
			},
		)
	}
	// If there's no candidate, it means no child starts before or during
	// the frozen frame and ends during or after the frozen frame.
	// This is not the call tree we're looking for.
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].durationNS > candidates[j].durationNS
	})
	// We pick the child with the longest duration and continue to look for
	// the leaf.
	return findFrameDropFrame(
		n.Children[candidates[0].index],
		frozenFrameStartNS,
		frozenFrameEndNS,
		st,
	)
}
