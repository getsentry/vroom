package nodetree

import (
	"hash"
)

type Node struct {
	DurationNS    uint64  `json:"duration_ns"`
	EndNS         uint64  `json:"-"`
	Fingerprint   uint64  `json:"fingerprint"`
	IsApplication bool    `json:"is_application"`
	Line          uint32  `json:"line,omitempty"`
	Name          string  `json:"name"`
	Package       string  `json:"package,omitempty"`
	Path          string  `json:"path,omitempty"`
	StartNS       uint64  `json:"-"`
	Children      []*Node `json:"children,omitempty"`
}

func NodeFromFrame(pkg, name, path string, line uint32, start, end, fingerprint uint64, isApplication bool) *Node {
	n := Node{
		EndNS:         end,
		Fingerprint:   fingerprint,
		IsApplication: isApplication,
		Line:          line,
		Name:          name,
		Package:       pkg,
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

func (n *Node) WriteToHash(h hash.Hash) {
	if n.Package == "" && n.Name == "" {
		h.Write([]byte("-"))
	} else {
		h.Write([]byte(n.Package))
		h.Write([]byte(n.Name))
	}
}
