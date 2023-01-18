package occurrence

import (
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
)

func Find(p profile.Profile, callTrees map[uint64][]*nodetree.Node) []Occurrence {
	var occurrences []Occurrence
	jobs, exists := detectExactFrameMetadata[p.Platform()]
	if exists {
		for _, metadata := range jobs {
			detectExactFrame(p, callTrees, metadata, &occurrences)
		}
	}
	return occurrences
}
