package nodetree

import (
	"github.com/getsentry/vroom/internal/calltree"
)

type Node struct {
	DurationNS    uint64  `json:"duration_ns"`
	EndNS         uint64  `json:"-"`
	Fingerprint   uint64  `json:"fingerprint"`
	ID            uint64  `json:"-"`
	IsApplication bool    `json:"is_application"`
	Line          uint32  `json:"line,omitempty"`
	Name          string  `json:"name"`
	Package       string  `json:"package"`
	Path          string  `json:"path,omitempty"`
	StartNS       uint64  `json:"-"`
	Children      []*Node `json:"children,omitempty"`
}

func NodeFromFrame(pkg, name, path string, line uint32, start, end, id uint64, isApplication bool) *Node {
	n := Node{
		EndNS:         end,
		Fingerprint:   id,
		ID:            id,
		IsApplication: isApplication,
		Line:          line,
		Name:          name,
		Package:       calltree.ImageBaseName(pkg),
		Path:          path,
		StartNS:       start,
	}
	if end > 0 {
		n.DurationNS = n.EndNS - n.StartNS
	}
	return &n
}

func (n *Node) SetDuration(t uint64) {
	n.EndNS = t
	n.DurationNS = n.EndNS - n.StartNS
}
