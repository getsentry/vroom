package calltree

// BacktraceAggregator performs aggregation of call trees from a stream of backtrace
// data rows. Call NewBacktraceAggregator() to create one, call Update() for each
// row, and then Finalize() once you're done adding data.
type BacktraceAggregatorP struct {
	// state per-transaction
	currentProfileID     string
	traceThreadCallTrees map[uint64]*CallTreeP   // thread ID -> call tree
	traceAllCallTrees    map[uint64][]*CallTreeP // thread ID -> call tree

	// global state
	ProfileIDToCallTreeInfo map[string]map[uint64][]*CallTreeP // profile ID -> thread ID -> call trees
	IsFinalized             bool
}

// Backtrace contains attributes for a single backtrace "row", which is a call
// stack capture for a single thread at a given point in time.
type BacktraceP struct {
	Addresses    []string
	IsMainThread bool
	ProfileID    string
	QueueName    string
	SessionKey   string
	ThreadID     uint64
	ThreadName   string
	TimestampNs  uint64
}

func NewBacktraceAggregatorP() *BacktraceAggregatorP {
	return &BacktraceAggregatorP{
		traceThreadCallTrees:    make(map[uint64]*CallTreeP),
		traceAllCallTrees:       make(map[uint64][]*CallTreeP),
		ProfileIDToCallTreeInfo: make(map[string]map[uint64][]*CallTreeP),
	}
}

func (a *BacktraceAggregatorP) Update(b BacktraceP) {
	if a.IsFinalized {
		panic("calltree: cannot call Update() after Finalize()")
	}
	if len(b.Addresses) == 0 {
		return
	}
	if a.currentProfileID != "" && a.currentProfileID != b.ProfileID {
		for _, tree := range a.traceThreadCallTrees {
			a.traceAllCallTrees[tree.ThreadID] = append(a.traceAllCallTrees[tree.ThreadID], tree)
		}
		a.traceThreadCallTrees = make(map[uint64]*CallTreeP)
		a.ProfileIDToCallTreeInfo[a.currentProfileID] = a.traceAllCallTrees
		a.traceAllCallTrees = make(map[uint64][]*CallTreeP)
	}
	a.currentProfileID = b.ProfileID

	// Reverse the addresses
	for i := len(b.Addresses)/2 - 1; i >= 0; i-- {
		opp := len(b.Addresses) - 1 - i
		b.Addresses[i], b.Addresses[opp] = b.Addresses[opp], b.Addresses[i]
	}

	callTree, ok := a.traceThreadCallTrees[b.ThreadID]
	if !ok {
		// There is no existing call tree for this thread to append to, start a new one
		newCallTree := backtraceToCallTreeP(b)
		if newCallTree != nil {
			a.traceThreadCallTrees[b.ThreadID] = newCallTree
		}
	} else if b.Addresses[0] != callTree.Address {
		// The previous call tree is complete, start a new one
		setEndTimeRecursiveP(callTree, b.TimestampNs)
		a.traceAllCallTrees[b.ThreadID] = append(a.traceAllCallTrees[b.ThreadID], callTree)
		delete(a.traceThreadCallTrees, b.ThreadID)

		newCallTree := backtraceToCallTreeP(b)
		if newCallTree != nil {
			a.traceThreadCallTrees[b.ThreadID] = newCallTree
		}
	} else {
		// This backtrace corresponds to the previous call tree for this thread
		current := callTree
		for _, address := range b.Addresses[1:] {
			var otherChildren []*CallTreeP
			var newCurrent *CallTreeP
			for _, child := range current.Children {
				if child.Address == address && child.EndTimeNs == NoEndTime {
					newCurrent = child
				} else {
					otherChildren = append(otherChildren, child)
				}
			}
			if newCurrent != nil {
				for _, child := range otherChildren {
					setEndTimeRecursiveP(child, b.TimestampNs)
				}
				current = newCurrent
			} else {
				for _, child := range current.Children {
					setEndTimeRecursiveP(child, b.TimestampNs)
				}
				newCallTree := &CallTreeP{
					ProfileID:    b.ProfileID,
					SessionKey:   b.SessionKey,
					Address:      address,
					ThreadID:     b.ThreadID,
					IsMainThread: b.IsMainThread,
					StartTimeNs:  b.TimestampNs,
					EndTimeNs:    NoEndTime,
					SelfTimeNs:   0,
				}
				current.Children = append(current.Children, newCallTree)
				current = newCallTree
			}
		}
		for _, child := range current.Children {
			setEndTimeRecursiveP(child, b.TimestampNs)
		}
	}
}

// Finalize should be called before accessing TraceIDToCallTreeInfo, otherwise
// the data may be incomplete.
func (a *BacktraceAggregatorP) Finalize() {
	if !a.IsFinalized && len(a.traceThreadCallTrees) > 0 && a.currentProfileID != "" {
		for _, tree := range a.traceThreadCallTrees {
			a.traceAllCallTrees[tree.ThreadID] = append(a.traceAllCallTrees[tree.ThreadID], tree)
		}
		a.ProfileIDToCallTreeInfo[a.currentProfileID] = a.traceAllCallTrees
		a.traceThreadCallTrees = nil
		a.traceAllCallTrees = nil
		a.currentProfileID = ""
	}
	a.IsFinalized = true
}

func backtraceToCallTreeP(b BacktraceP) *CallTreeP {
	threadName := getCallTreeThreadName(b.QueueName, b.ThreadName)
	root := &CallTreeP{
		Address:      b.Addresses[0],
		EndTimeNs:    NoEndTime,
		IsMainThread: b.IsMainThread,
		ProfileID:    b.ProfileID,
		SessionKey:   b.SessionKey,
		StartTimeNs:  b.TimestampNs,
		ThreadID:     b.ThreadID,
		ThreadName:   threadName,
	}
	current := root
	for _, address := range b.Addresses[1:] {
		tree := &CallTreeP{
			ProfileID:   b.ProfileID,
			SessionKey:  b.SessionKey,
			Address:     address,
			ThreadID:    b.ThreadID,
			ThreadName:  threadName,
			StartTimeNs: b.TimestampNs,
			EndTimeNs:   NoEndTime,
		}
		current.Children = append(current.Children, tree)
		current = tree
	}
	return root
}

func setEndTimeRecursiveP(root *CallTreeP, timestampNs uint64) {
	if root.EndTimeNs != NoEndTime {
		return
	}
	root.EndTimeNs = timestampNs
	var totalChildDuration uint64
	for _, child := range root.Children {
		setEndTimeRecursiveP(child, timestampNs)
		childDuration := TotalDurationP(child)
		if child.EndTimeNs != NoEndTime {
			totalChildDuration += childDuration
		}
	}
	rootDuration := TotalDurationP(root)
	if root.EndTimeNs != NoEndTime {
		root.SelfTimeNs = rootDuration - totalChildDuration
	}
}

func getCallTreeThreadName(queueName, threadName string) string {
	if queueName != "" {
		return queueName
	} else {
		return threadName
	}
}

// TotalDuration returns the total duration of the call tree and all of its
// child trees. Returns `NoEndTime` if the tree has no end time and a duration
// cannot be computed.
func TotalDurationP(callTree *CallTreeP) uint64 {
	if callTree.EndTimeNs == NoEndTime {
		return 0
	}
	return callTree.EndTimeNs - callTree.StartTimeNs
}
