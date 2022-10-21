package nodetree

import (
	"hash"
	"path"
)

type Node struct {
	DurationNS    uint64  `json:"duration_ns"`
	EndNS         uint64  `json:"-"`
	Fingerprint   uint64  `json:"fingerprint"`
	IsApplication bool    `json:"is_application"`
	Line          uint32  `json:"line,omitempty"`
	Name          string  `json:"name"`
	Package       string  `json:"package"`
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
		Package:       PackageBaseName(pkg),
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

// PackageBaseName returns the basename of the package if package is a path.
func PackageBaseName(p string) string {
	if p == "" {
		return ""
	}
	return path.Base(p)
}

func (n *Node) Collapse() *Node {
	for i, child := range n.Children {
		n.Children[i] = child.Collapse()
	}

	// If the only child runs for the entirety of the parent,
	// we want to collapse them by taking the inner most application frame.
	// If neither are application frames, we take the inner most frame
	if len(n.Children) == 1 && n.StartNS == child.StartNS && n.DurationNS == child.DurationNS {
		if child.IsApplication {
			n = child
		} else if n.IsApplication {
			n.Children = child.Children
		} else {
			n = child
		}
	}

	return n
}
