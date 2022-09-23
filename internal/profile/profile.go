package profile

import "github.com/getsentry/vroom/internal/nodetree"

type (
	Profile interface {
		GetID() string
		GetOrganizationID() uint64
		GetProjectID() uint64

		CallTrees() (map[uint64][]*nodetree.Node, error)
		StoragePath() string
	}
)
