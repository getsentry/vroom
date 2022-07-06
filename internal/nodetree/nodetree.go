package nodetree

import (
	"encoding/binary"
	"hash"

	"github.com/getsentry/vroom/internal/calltree"
)

type Node struct {
	DurationNS uint64  `json:"duration_ns"`
	EndNS      uint64  `json:"-"`
	ID         uint64  `json:"id"`
	Line       uint32  `json:"line,omitempty"`
	Name       string  `json:"name"`
	Package    string  `json:"package"`
	Path       string  `json:"path,omitempty"`
	StartNS    uint64  `json:"-"`
	Children   []*Node `json:"children,omitempty"`
}

func NodeFromFrame(pkg, name, path string, line uint32, parent, timestamp uint64, id uint64) *Node {
	n := Node{
		EndNS:   timestamp,
		ID:      id,
		Name:    name,
		Package: calltree.ImageBaseName(pkg),
		Path:    path,
		Line:    line,
		StartNS: parent,
	}
	n.DurationNS = n.EndNS - n.StartNS
	return &n
}

func (n *Node) SetDuration(t uint64) {
	n.EndNS = t
	n.DurationNS = n.EndNS - n.StartNS
}

func (n *Node) Wraps(v *Node) bool {
	return n.StartNS <= v.StartNS && v.EndNS <= n.EndNS
}

func (n *Node) Insert(v *Node, fingerprint hash.Hash64) {
	buffer := make([]byte, 8)
	binary.LittleEndian.PutUint64(buffer, v.ID)
	fingerprint.Write(buffer)
	for _, c := range n.Children {
		if c.Wraps(v) {
			c.Insert(v, fingerprint)
			return
		}
	}
	v.ID = fingerprint.Sum64()
	n.Children = append(n.Children, v)
}
