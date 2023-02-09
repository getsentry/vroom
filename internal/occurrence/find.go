package occurrence

import (
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
)

func Find(p profile.Profile, callTrees map[uint64][]*nodetree.Node) []*Occurrence {
	var occurrences []*Occurrence
	if jobs, exists := detectFrameJobs[p.Platform()]; exists {
		for _, metadata := range jobs {
			detectFrame(p, callTrees, metadata, &occurrences)
		}
	}
	return occurrences
}
