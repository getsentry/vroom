package nodetree

import (
	"hash"

	"github.com/getsentry/vroom/internal/frame"
)

type (
	Node struct {
		Children      []*Node `json:"children,omitempty"`
		DurationNS    uint64  `json:"duration_ns"`
		Fingerprint   uint64  `json:"fingerprint"`
		IsApplication bool    `json:"is_application"`
		Line          uint32  `json:"line,omitempty"`
		Name          string  `json:"name"`
		Package       string  `json:"package"`
		Path          string  `json:"path,omitempty"`

		EndNS       uint64      `json:"-"`
		Frame       frame.Frame `json:"-"`
		SampleCount int         `json:"-"`
		StartNS     uint64      `json:"-"`
	}
)

func NodeFromFrame(f frame.Frame, start, end, fingerprint uint64) *Node {
	inApp := true
	if f.InApp != nil {
		inApp = *f.InApp
	}
	n := Node{
		EndNS:         end,
		Fingerprint:   fingerprint,
		Frame:         f,
		IsApplication: inApp,
		Line:          f.Line,
		Name:          f.Function,
		Package:       f.PackageBaseName(),
		Path:          f.Path,
		SampleCount:   1,
		StartNS:       start,
	}
	if end > 0 {
		n.DurationNS = n.EndNS - n.StartNS
	}
	return &n
}

func (n *Node) Update(timestamp uint64) {
	n.SampleCount++
	n.SetDuration(timestamp)
}

func (n *Node) ToFrame() frame.Frame {
	n.Frame.Data.SymbolicatorStatus = n.Frame.Status
	return n.Frame
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

func (n Node) Collapse() []*Node {
	// If a node has insufficient sample count, it likely isn't interesting to
	// surface as a suspect function because we only saw it for a moment. So
	// we only consider nodes that appear for more than 1 consecutive samples.
	if n.SampleCount <= 1 {
		return []*Node{}
	}

	// always collapse the children first, since pruning may reduce
	// the number of children
	children := make([]*Node, 0, len(n.Children))
	for _, child := range n.Children {
		children = append(children, child.Collapse()...)
	}
	n.Children = children

	// If the current node is an unknown frame, we just return
	// its children. The children are guaranteed not to be
	// unknown nodes since they made it through a `.Collapse`
	// call earlier already
	if n.Name == "" {
		return n.Children
	}

	// If the only child runs for the entirety of the parent,
	// we want to collapse them by taking the inner most application frame.
	// If neither are application frames, we take the inner most frame
	if len(n.Children) == 1 {
		child := n.Children[0]
		if n.StartNS == child.StartNS && n.DurationNS == child.DurationNS {
			if n.IsApplication {
				if child.IsApplication {
					// if the node and it's child are both application frames,
					// we only want the inner one
					n = *child
				} else {
					// if the node is an application frame but the child is not,
					// we want to skip the child frame
					n.Children = child.Children
				}
			} else {
				// if the node is not an application frame,
				// we want to skip it and favour it's child
				n = *child
			}
		}
	}

	return []*Node{&n}
}
