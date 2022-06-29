package calltree

import (
	"crypto/md5"
	"fmt"
	"hash"
	"io"

	"github.com/getsentry/vroom/internal/errorutil"
)

var errDataIntegrityNilTree = fmt.Errorf("calltree: %w: tree must be non-nil", errorutil.ErrDataIntegrity)

// AggregateCallTree represents a call tree that represents the aggregation of
// multiple instances of the same call tree pattern across many traces.
type AggregateCallTree struct {
	Image            string
	Symbol           string
	DemangledSymbol  string
	Line             uint32
	Path             string
	Package          string
	TotalDurationsNs []float64
	SelfDurationsNs  []float64
	Children         []*AggregateCallTree
}

// path is a slice of indices that describes a path through a call tree. The first
// element of this slice corresponds to the first child of the root node.
type treePath []int

func (act *AggregateCallTree) identifier() string {
	return ImageBaseName(act.Image) + ":" + act.Symbol
}

// shallowCopy returns a copy of the call tree without copying its children.
func (act *AggregateCallTree) shallowCopy() *AggregateCallTree {
	var totalDurationsCopy []float64
	if act.TotalDurationsNs != nil {
		totalDurationsCopy = make([]float64, len(act.TotalDurationsNs))
		copy(totalDurationsCopy, act.TotalDurationsNs)
	}

	var selfDurationsCopy []float64
	if act.SelfDurationsNs != nil {
		selfDurationsCopy = make([]float64, len(act.SelfDurationsNs))
		copy(selfDurationsCopy, act.SelfDurationsNs)
	}

	return &AggregateCallTree{
		Image:            act.Image,
		Symbol:           act.Symbol,
		DemangledSymbol:  act.DemangledSymbol,
		Line:             act.Line,
		Path:             act.Path,
		Package:          act.Package,
		TotalDurationsNs: totalDurationsCopy,
		SelfDurationsNs:  selfDurationsCopy,
		Children:         nil, // shallow copy, not copying children
	}
}

// deepCopy returns a recursive copy of the call tree.
func (act *AggregateCallTree) deepCopy() *AggregateCallTree {
	clone := act.shallowCopy()
	if len(act.Children) == 0 {
		return clone
	}
	clone.Children = make([]*AggregateCallTree, 0, len(act.Children))
	for _, child := range act.Children {
		clone.Children = append(clone.Children, child.deepCopy())
	}
	return clone
}

// pathCopy returns a copy of the call tree by *only* copying the nodes
// included in the specified path up until the end of the path, and then
// performing a regular deep copy for the remaining children.
func (act *AggregateCallTree) pathCopy(p treePath) *AggregateCallTree {
	if len(p) == 0 {
		return act.deepCopy()
	}
	clone := act.shallowCopy()
	head, rest := p[0], p[1:]
	child := act.Children[head].pathCopy(rest)
	clone.Children = append(clone.Children, child)
	return clone
}

// shallowHash computes a hash of the call tree without including its children.
func (act *AggregateCallTree) shallowHash(h hash.Hash) {
	// Use placeholders in place of empty strings, because otherwise
	// we could be generating the same hash for two different call tree
	// structures. Take this example:
	//
	// A -> unknown -> B
	// A -> B
	//
	// If the unknown node is hashed as an empty string, then that first
	// call tree will have the same hash as the second one, even though
	// they are not the same.
	image := ImageBaseName(act.Image)
	if image == "" {
		image = "$i"
	}
	symbol := act.Symbol
	if symbol == "" {
		symbol = "$s"
	}

	_, _ = io.WriteString(h, image)
	_, _ = io.WriteString(h, symbol)
}

// deepHash computes a hash of the call tree recursively.
func (act *AggregateCallTree) deepHash(h hash.Hash) {
	act.shallowHash(h)
	for _, child := range act.Children {
		child.deepHash(h)
	}
}

// pathHash computes a hash of the nodes in the specified path.
func (act *AggregateCallTree) pathHash(p treePath, h hash.Hash) {
	act.shallowHash(h)
	if len(p) == 0 {
		return
	}
	head, rest := p[0], p[1:]
	act.Children[head].pathHash(rest, h)
}

// shallowMerge merges another tree into the current tree, without
// merging the children.
func (act *AggregateCallTree) shallowMerge(other *AggregateCallTree) {
	if act == other {
		return
	}

	if act.DemangledSymbol == "" {
		act.DemangledSymbol = other.DemangledSymbol
	}
	// Paths and line numbers can change as the source code of an application
	// changes, so always bias toward the newer value when merging.
	if other.Path != "" {
		act.Path = other.Path
		act.Line = other.Line
	}
	act.TotalDurationsNs = append(act.TotalDurationsNs, other.TotalDurationsNs...)
	act.SelfDurationsNs = append(act.SelfDurationsNs, other.SelfDurationsNs...)
}

// deepMerge merges another tree into the current tree, recursively.
func (act *AggregateCallTree) deepMerge(other *AggregateCallTree) {
	act.shallowMerge(other)
	identifierToChildMap := make(map[string]*AggregateCallTree, len(act.Children))
	for _, child := range act.Children {
		identifierToChildMap[child.identifier()] = child
	}
	newChildren := make([]*AggregateCallTree, 0, len(other.Children))
	for _, otherChild := range other.Children {
		if child, ok := identifierToChildMap[otherChild.identifier()]; ok {
			child.deepMerge(otherChild)
		} else {
			newChildren = append(newChildren, otherChild)
		}
	}
	act.Children = append(act.Children, newChildren...)
}

// pathMerge merges another tree into the current tree by *only* merging the
// nodes included in the specified path up until the end of the path, and then
// performing a regular deep merge for the remaining children.
func (act *AggregateCallTree) pathMerge(other *AggregateCallTree, p treePath) {
	if len(p) == 0 {
		act.deepMerge(other)
		return
	}
	act.shallowMerge(other)
	head, rest := p[0], p[1:]
	otherChild := other.Children[head]
	for _, child := range act.Children {
		if IsImageEqual(child.Image, otherChild.Image) && child.Symbol == otherChild.Symbol {
			child.pathMerge(otherChild, rest)
			break
		}
	}
}

func (act *AggregateCallTree) Symbols() []string {
	uniqueSymbols := map[string]struct{}{
		act.Symbol: {},
	}

	for _, c := range act.Children {
		for _, s := range c.Symbols() {
			uniqueSymbols[s] = struct{}{}
		}
	}

	symbols := make([]string, 0, len(uniqueSymbols))

	for s := range uniqueSymbols {
		symbols = append(symbols, s)
	}

	return symbols
}

// CallTreeAggregator aggregates AggregateCallTree's and produces a map of unique
// root call trees, keyed by a hash that uniquely identifies the call tree pattern.
type CallTreeAggregator struct {
	UniqueRootCallTrees map[string]*AggregateCallTree
}

func NewCallTreeAggregator() *CallTreeAggregator {
	return &CallTreeAggregator{
		UniqueRootCallTrees: make(map[string]*AggregateCallTree),
	}
}

// Update merges new aggregate call tree data into the existing data accumulated
// by the aggregator. targetImage/targetSymbol (optional) identify a particular
// node to target, such that we compute the unique paths through the tree to the
// target nodes and merge those paths *independently*. If targetImage/targetSymbol
// are unspecified, then the entire tree will be merged as-is.
//
// Returns the keys of the unique root call trees that were created or updated.
func (a *CallTreeAggregator) Update(root *AggregateCallTree, targetImage, targetSymbol string) ([]string, error) {
	if root == nil {
		return nil, errDataIntegrityNilTree
	}

	// If no target node is specified, compute a hash over the entire tree. If
	// it exists, we merge it, otherwise consider it to be a new root node.
	if targetImage == "" && targetSymbol == "" {
		h := md5.New()
		root.deepHash(h)
		key := fmt.Sprintf("%x", h.Sum(nil))

		if existing, ok := a.UniqueRootCallTrees[key]; ok {
			existing.deepMerge(root)
		} else {
			a.UniqueRootCallTrees[key] = root.deepCopy()
		}

		return []string{key}, nil
	}

	// Find the set of paths through the tree that include the target node(s),
	// ignore all other paths.
	paths := findMatchingPaths(root, targetImage, targetSymbol)
	var allKeys []string
	for _, p := range paths {
		h := md5.New()
		root.pathHash(p, h)
		key := fmt.Sprintf("%x", h.Sum(nil))

		if existing, ok := a.UniqueRootCallTrees[key]; ok {
			existing.pathMerge(root, p)
		} else {
			a.UniqueRootCallTrees[key] = root.pathCopy(p)
		}

		allKeys = append(allKeys, key)
	}
	return allKeys, nil
}

// findMatchingPaths performs a depth first search on the tree starting at the
// specified root node, and searches for a node with a matching image and symbol.
// It returns all of the unique paths (slice of indices) through the tree to a
// node with a matching image and symbol.
func findMatchingPaths(root *AggregateCallTree, image, symbol string) []treePath {
	if root == nil {
		return nil
	}
	var paths []treePath
	_findMatchingPaths(root, image, symbol, &paths, nil)
	return paths
}

func _findMatchingPaths(root *AggregateCallTree, image, symbol string, paths *[]treePath, currentPath treePath) {
	// When we find the target image and symbol
	if IsImageEqual(root.Image, image) && root.Symbol == symbol {
		// We look if the call is recursive by checking the next level
		var recursiveCall bool
		for _, c := range root.Children {
			if IsImageEqual(c.Image, image) && c.Symbol == symbol {
				recursiveCall = true
				break
			}
		}

		// If it's not, we add it to the path, otherwise we'll add it later
		if !recursiveCall {
			*paths = append(*paths, currentPath)
		}
	}

	currentPathLength := len(currentPath)

	for index, child := range root.Children {
		path := make(treePath, currentPathLength+1)
		copy(path, currentPath)
		path[currentPathLength] = index
		_findMatchingPaths(child, image, symbol, paths, path)
	}
}
