package occurrence

import (
	"sort"
	"strings"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
)

type (
	inAppFrameInfo struct {
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
			var inAppFrames []inAppFrameInfo
			findAllInAppFramesInSection(
				root,
				mv.ElapsedSinceStartNs-uint64(mv.Value),
				mv.ElapsedSinceStartNs,
				&inAppFrames,
				&st,
			)
			// No in app frame in the section, we bail.
			if len(inAppFrames) == 0 {
				continue
			}
			// Order frames by duration descending and if equal durations,
			// choose the deepest frame.
			sort.SliceStable(inAppFrames, func(i, j int) bool {
				a, b := inAppFrames[i].n, inAppFrames[j].n
				if a.DurationNS == b.DurationNS {
					return i > j
				}
				return a.DurationNS > b.DurationNS
			})
			// This will be the deepest in app frame we think is responsible.
			iaf := refineCause(&inAppFrames[0])
			// We found a potential stacktrace responsible for this frozen frame
			stackTrace := make([]frame.Frame, 0, len(iaf.st))
			for _, f := range iaf.st {
				stackTrace = append(stackTrace, f.ToFrame())
			}
			*occurrences = append(
				*occurrences,
				NewOccurrence(p, nodeInfo{
					Category:   FrameDrop,
					Node:       *iaf.n,
					StackTrace: stackTrace,
				}),
			)
			break
		}
	}
}

func refineCause(iaf *inAppFrameInfo) *inAppFrameInfo {
	// If there are many children or none, we keep the current cause.
	if len(iaf.n.Children) != 1 {
		return iaf
	}
	// If there's only one children, try to find a deeper in app frame.
	// The in app frame must be of the same duration other we're not making
	// progress.
	c := iaf.n.Children[0]
	if iaf.n.DurationNS != c.DurationNS {
		return iaf
	}
	cause := refineCause(&inAppFrameInfo{
		iaf.depth + 1,
		c,
		append(iaf.st, c),
	})
	// If the frame returned is from the app, it's our new cause.
	if cause.n.IsApplication {
		return cause
	}
	return iaf
}

// findAllInAppFramesInSection will find all in app frames from the start of
// the frozen frame to the end of it and return at which depth we found each
// frame and an associated stack trace.
func findAllInAppFramesInSection(
	n *nodetree.Node,
	frozenFrameStartNS uint64,
	frozenFrameEndNS uint64,
	inAppFrames *[]inAppFrameInfo,
	st *[]*nodetree.Node,
) {
	*st = append(*st, n)
	defer func() {
		*st = (*st)[:len(*st)-1]
	}()
	if !strings.HasPrefix(n.Frame.Function, "@objc") &&
		n.StartNS >= frozenFrameStartNS &&
		n.EndNS <= frozenFrameEndNS &&
		n.IsApplication {
		*inAppFrames = append(*inAppFrames, inAppFrameInfo{
			0,
			n,
			*st,
		})
	}
	for _, c := range n.Children {
		findAllInAppFramesInSection(c, frozenFrameStartNS, frozenFrameEndNS, inAppFrames, st)
	}
}
