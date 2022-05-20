package aggregate

import (
	"fmt"
	"sort"

	"github.com/getsentry/vroom/internal/errorutil"
	"github.com/getsentry/vroom/internal/quantile"
)

// MergeAllCallTreesInBacktrace returns a single call tree for a single trace ID merging all call trees
// returned during aggregation. The aggregation will return a calltree per method key, which are the same call tree
// since there is only 1 trace, and we need to deduplicate them again to display them properly.
// This was done to still use the aggregation to produce the call tree and needs to be refactored
// to fully skip the aggregation step and just return the call tree from the trace
func MergeAllCallTreesInBacktrace(ba *Aggregate) ([]CallTree, error) {
	allCallTrees := make(map[string][]CallTree)
	for _, callTrees := range ba.FunctionToCallTrees {
		for _, callTree := range callTrees {
			allCallTrees[callTree.RootFrame.ID] = append(allCallTrees[callTree.RootFrame.ID], callTree)
		}
	}

	mergedCallTrees := make([]CallTree, 0, len(allCallTrees))
	for _, callTrees := range allCallTrees {
		mergedCallTree, err := mergeCallTrees(callTrees)
		if err != nil {
			return nil, err
		}
		mergedCallTree.ProfileIDs = mergedCallTree.ProfileIDs[:1]
		mergedCallTrees = append(mergedCallTrees, mergedCallTree)
	}
	sortCallTrees(mergedCallTrees)
	return mergedCallTrees, nil
}

func mergeCallTrees(pbs []CallTree) (CallTree, error) {
	if len(pbs) == 0 {
		return CallTree{}, nil
	}
	head, rest := pbs[0], pbs[1:]
	var restRootFrames []Frame
	for _, tree := range rest {
		head.Count += tree.Count
		for threadName, count := range tree.ThreadNameToCount {
			head.ThreadNameToCount[threadName] += count
		}
		head.ProfileIDs = append(head.ProfileIDs, tree.ProfileIDs...)
		restRootFrames = append(restRootFrames, tree.RootFrame)
	}
	sort.Strings(head.ProfileIDs)
	if err := mergeCallTreeFrames(&head.RootFrame, restRootFrames); err != nil {
		return CallTree{}, err
	}
	return head, nil
}

func mergeCallTreeFrames(head *Frame, rest []Frame) error {
	for _, frame := range rest {
		if g, w := head.Identifier(), frame.Identifier(); g != w {
			return fmt.Errorf("backtrace: %w: trying to merge nodes with different identifiers: head has %q, frame has %q", errorutil.ErrDataIntegrity, g, w)
		}
		if frame.Path != "" {
			head.Path = frame.Path
			head.Line = frame.Line
		}
		head.TotalDurationNsValues = append(head.TotalDurationNsValues, frame.TotalDurationNsValues...)
		head.SelfDurationNsValues = append(head.SelfDurationNsValues, frame.SelfDurationNsValues...)
		frame.TotalDurationNsValues = nil
		frame.SelfDurationNsValues = nil
	}

	head.TotalDurationNs = quantileToAggQuantiles(quantile.Quantile{Xs: head.TotalDurationNsValues})
	head.SelfDurationNs = quantileToAggQuantiles(quantile.Quantile{Xs: head.SelfDurationNsValues})
	head.TotalDurationNsValues = nil
	head.SelfDurationNsValues = nil

	mergeableChildren := make(map[string][]Frame, len(head.Children))
	for _, child := range head.Children {
		id := child.Identifier()
		mergeableChildren[id] = append(mergeableChildren[id], child)
	}
	for _, frame := range rest {
		for _, child := range frame.Children {
			id := child.Identifier()
			mergeableChildren[id] = append(mergeableChildren[id], child)
		}
	}
	for _, children := range mergeableChildren {
		if len(children) == 0 {
			continue
		}
		headChild, restChildren := children[0], children[1:]
		if err := mergeCallTreeFrames(&headChild, restChildren); err != nil {
			return err
		}
		// This could be a node that exists in rest, but not in head, so it
		// has to be added to head's children if that is the case.
		headChildAlreadyInChildren := false
		for _, child := range head.Children {
			if &child == &headChild {
				headChildAlreadyInChildren = true
				break
			}
		}
		if !headChildAlreadyInChildren {
			head.Children = append(head.Children, headChild)
		}
	}
	return nil
}
