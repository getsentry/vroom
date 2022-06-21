package aggregate

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/getsentry/vroom/internal/calltree"
	"github.com/getsentry/vroom/internal/quantile"
	"github.com/getsentry/vroom/internal/snubautil"
)

// The list of symbols that indicate a thread waiting is from:
// https://github.com/brendangregg/FlameGraph/blob/1a0dc6985aad06e76857cf2a354bd5ba0c9ce96b/stackcollapse-sample.awk#L72
var waitingSymbols = map[string]bool{
	"0x0":                              true,
	"_sigtramp":                        true,
	"__psynch_cvwait":                  true,
	"__select":                         true,
	"__semwait_signal":                 true,
	"__ulock_wait":                     true,
	"kevent":                           true,
	"mach_msg_trap":                    true,
	"read":                             true,
	"semaphore_wait_trap":              true,
	"_dispatch_worker_thread2":         true,
	"_dispatch_workloop_invoke2":       true,
	"_dispatch_workloop_worker_thread": true,
	"_dispatch_client_callout":         true,
	"_dispatch_client_callout2":        true,
	"__CFRunLoopRun":                   true,
	"__CFRUNLOOP_IS_CALLING_OUT_TO_AN_OBSERVER_CALLBACK_FUNCTION__": true,
	"start_wqthread":     true,
	"__workq_kernreturn": true,
}

type (
	BacktraceAggregatorP struct {
		bta                        *calltree.BacktraceAggregatorP
		n                          int
		profileIDToTransactionName map[string]string
		symbolsByProfileID         map[string]map[string]Symbol
	}

	sessionDataP struct {
		key           string
		functionCalls []functionCallP
	}

	functionDataP struct {
		count               int
		durationsNs         quantile.Quantile
		image               string
		line                int
		mainThreadCount     int
		path                string
		profileIDToCount    map[string]int
		profileIDToThreadID map[string]uint64
		symbol              string
		threadNames         map[string]int
		uniqueCallTrees     map[*calltree.CallTreeP]bool
	}

	functionCallWithDurationP struct {
		key           string
		data          *functionDataP
		durationNsP75 float64
	}

	functionCallP struct {
		address      string
		durationNs   uint64
		isMainThread bool
		profileID    string
		rootCallTree *calltree.CallTreeP
		threadID     uint64
		threadName   string
	}
)

// The default maximum number of exemplar traces to select from each set.
// TODO: figure out how to intelligently select exemplars that are more representative of the entire population.
const defaultNExemplarTraces = 25

func (a *BacktraceAggregatorP) SetTopNFunctions(n int) {
	a.n = n
}

func (a *BacktraceAggregatorP) UpdateFromProfile(profile snubautil.Profile) error {
	var iosProfile IosProfile
	err := json.Unmarshal([]byte(profile.Profile), &iosProfile)
	if err != nil {
		return err
	}
	a.profileIDToTransactionName[profile.ProfileID] = profile.TransactionName
	for _, sample := range iosProfile.Samples {
		onMainThread := sample.ContainsMain()
		queueMetadata, qmExists := iosProfile.QueueMetadata[sample.QueueAddress]
		// Skip samples with a queue called "com.apple.main-thread"
		// but not being scheduled on what we detected as the main thread.
		if queueMetadata.IsMainThread() && !onMainThread {
			continue
		}
		threadID := strconv.FormatUint(sample.ThreadID, 10)
		threadMetadata := iosProfile.ThreadMetadata[threadID]
		threadName := threadMetadata.Name
		if threadName == "" && qmExists {
			threadName = queueMetadata.Label
		} else {
			threadName = threadID
		}
		addresses := make([]string, len(sample.Frames), len(sample.Frames))
		for i, frame := range sample.Frames {
			addresses[i] = frame.InstructionAddr

			var symbolName, imageName string
			if frame.Function != "" {
				symbolName = frame.Function
				imageName = calltree.ImageBaseName(frame.Package)
			}
			if symbolName == "" {
				symbolName = fmt.Sprintf("unknown (%s)", frame.InstructionAddr)
			}
			symbol := Symbol{
				Image:    imageName,
				Name:     symbolName,
				Filename: frame.Filename,
				Path:     frame.Package,
				Line:     frame.LineNo,
			}
			if _, exists := a.symbolsByProfileID[profile.ProfileID]; !exists {
				a.symbolsByProfileID[profile.ProfileID] = make(map[string]Symbol)
			}
			a.symbolsByProfileID[profile.ProfileID][frame.InstructionAddr] = symbol
		}
		a.bta.Update(calltree.BacktraceP{
			ProfileID:    profile.ProfileID,
			Addresses:    addresses,
			IsMainThread: onMainThread,
			ThreadID:     sample.ThreadID,
			ThreadName:   threadName,
			TimestampNs:  sample.RelativeTimestampNS,
		})
	}

	return nil
}

func (a *BacktraceAggregatorP) Result() (Aggregate, error) {
	a.bta.Finalize()

	profileIDToSessionData := make(map[string]sessionDataP)
	profileIDs := make([]string, 0, len(a.bta.ProfileIDToCallTreeInfo))

	// Iterate through the list of profiles and group the addresses to symbolicate
	// by their session key, so that we can run batch symbolication.
	for profileID, threadIDToCallTrees := range a.bta.ProfileIDToCallTreeInfo {
		profileIDs = append(profileIDs, profileID)
		functionCalls := make([]functionCallP, 0)
		var currentSessionKey string
		for _, callTrees := range threadIDToCallTrees {
			for _, callTree := range callTrees {
				accumulateFunctionCallsP(callTree, callTree, profileID, &functionCalls)
				if currentSessionKey == "" {
					currentSessionKey = callTree.SessionKey
				} else if currentSessionKey != callTree.SessionKey {
					return Aggregate{}, fmt.Errorf("backtrace: unexpected multiple session keys in the same trace: %q and %q", currentSessionKey, callTree.SessionKey)
				}
			}
		}
		profileIDToSessionData[profileID] = sessionDataP{
			key:           profileID,
			functionCalls: functionCalls,
		}
	}

	if len(a.symbolsByProfileID) == 0 {
		return Aggregate{}, nil
	}

	// Iterate through all calls from all traces and bucket them by image and symbol
	bucketedFunctionData := make(map[string]*functionDataP)
	for profileID, sessionData := range profileIDToSessionData {
		symbols, ok := a.symbolsByProfileID[profileID]
		if !ok {
			continue
		}
		for _, call := range sessionData.functionCalls {
			symbol, ok := symbols[call.address]
			if !ok {
				continue
			}
			if waitingSymbols[symbol.Name] {
				continue
			}
			h := computeFunctionHash(symbol.Image, symbol.Name)
			key := fmt.Sprintf("%x", h.Sum(nil))
			data, ok := bucketedFunctionData[key]
			if !ok {
				data = &functionDataP{
					image:               symbol.Image,
					symbol:              symbol.Name,
					threadNames:         make(map[string]int),
					profileIDToCount:    make(map[string]int),
					profileIDToThreadID: make(map[string]uint64),
					uniqueCallTrees:     make(map[*calltree.CallTreeP]bool),
				}
				bucketedFunctionData[key] = data
			}
			data.profileIDToThreadID[profileID] = call.threadID
			data.threadNames[call.threadName]++
			data.durationsNs.Add(float64(call.durationNs))
			data.profileIDToCount[profileID]++
			data.count++
			if call.isMainThread {
				data.mainThreadCount++
			}
			if data.path == "" {
				if line, path, ok := symbol.GetPath(); ok {
					data.line = line
					data.path = path
				}
			}
			data.uniqueCallTrees[call.rootCallTree] = true
		}
	}

	// Figure out which ones are the top functions
	var functionsWithDurations []functionCallWithDurationP
	for key, data := range bucketedFunctionData {
		if strings.HasPrefix(data.symbol, "unknown") {
			continue
		}
		functionsWithDurations = append(functionsWithDurations, functionCallWithDurationP{
			key:           key,
			data:          data,
			durationNsP75: data.durationsNs.Percentile(0.75),
		})
	}

	sort.SliceStable(functionsWithDurations, func(i, j int) bool {
		iFwd, jFwd := functionsWithDurations[i], functionsWithDurations[j]
		if iFwd.durationNsP75 != jFwd.durationNsP75 {
			if math.IsNaN(iFwd.durationNsP75) {
				return false
			}
			if math.IsNaN(jFwd.durationNsP75) {
				return true
			}
			return jFwd.durationNsP75 < iFwd.durationNsP75
		}
		return iFwd.key < jFwd.key
	})

	topFunctionsCount := min(len(functionsWithDurations), a.n)
	topFunctions := functionsWithDurations[:topFunctionsCount]

	// Calculate aggregate statistics for every unique function call
	aggregateCalls := make([]FunctionCall, 0, topFunctionsCount)
	for _, call := range topFunctions {
		data := call.data
		frequency := make([]float64, 0, len(data.profileIDToCount))
		profileIDs := make([]string, 0, len(data.profileIDToCount))
		var interactions []string
		uniqueInteractions := make(map[string]struct{})
		for profileID, count := range data.profileIDToCount {
			frequency = append(frequency, float64(count))
			profileIDs = append(profileIDs, profileID)
			i, exists := a.profileIDToTransactionName[profileID]
			if !exists {
				continue
			}
			if _, exists := uniqueInteractions[i]; exists {
				continue
			}
			interactions = append(interactions, i)
			uniqueInteractions[i] = struct{}{}
		}
		// Add zero frequencies for every trace ID that did not contain
		// any occurrences of this function call.
		frequency = append(frequency, make([]float64, len(profileIDToSessionData)-len(data.profileIDToCount))...)
		totalCount := float32(data.count)
		threadNameToPercent := make(map[string]float32)
		for name, count := range data.threadNames {
			threadNameToPercent[name] = float32(count) / totalCount
		}
		sort.Strings(profileIDs)
		aggregateCalls = append(aggregateCalls, FunctionCall{
			Image:               data.image,
			Symbol:              data.symbol,
			DurationNs:          quantileToAggQuantiles(data.durationsNs),
			Frequency:           quantileToAggQuantiles(quantile.Quantile{Xs: frequency}),
			MainThreadPercent:   float32(data.mainThreadCount) / totalCount,
			ThreadNameToPercent: threadNameToPercent,
			Line:                data.line,
			Path:                data.path,
			ProfileIDs:          profileIDs,
			ProfileIDToThreadID: data.profileIDToThreadID,
			TransactionNames:    interactions,
			Key:                 call.key,
		})
	}

	functionToCallTrees, err := a.computeFunctionsToCallTreesMap(topFunctions)
	if err != nil {
		return Aggregate{}, err
	}

	return Aggregate{
		FunctionCalls:       aggregateCalls,
		FunctionToCallTrees: functionToCallTrees,
	}, nil

}

func (a *BacktraceAggregatorP) computeFunctionsToCallTreesMap(fns []functionCallWithDurationP) (map[string][]CallTree, error) {
	// Compute the call trees for each function separately
	functionToCallTrees := make(map[string][]CallTree)
	for _, fn := range fns {
		trees, err := a.computeCallTreesForFunctionP(fn, a.symbolsByProfileID)
		if err != nil {
			return nil, err
		}
		functionToCallTrees[fn.key] = trees
	}

	return functionToCallTrees, nil
}

func (a *BacktraceAggregatorP) computeCallTreesForFunctionP(f functionCallWithDurationP, symbolsByProfileID map[string]map[string]Symbol) ([]CallTree, error) {
	// Create aggregate call trees using the symbolication results, then use
	// a CallTreeAggregator to unique them.
	agg := calltree.NewCallTreeAggregator()
	// Map of call tree key -> thread name -> count for that thread
	treeKeyToThreadCounts := make(map[string]map[string]uint64)
	// Map of call tree key -> unique trace IDs that include that call tree
	treeKeyToprofileIDs := make(map[string]map[string]bool)
	for tree := range f.data.uniqueCallTrees {
		var rootAct *calltree.AggregateCallTree
		// Filter out a bogus root address that appears in some iOS backtraces, this symbol
		// can never be symbolicated and usually contains 1 child. It looks like this
		// in the backtrace:
		//
		// ...
		// 25  libsystem_pthread.dylib             0x00007fff60c8e498 _pthread_wqthread + 313
		// 26  libsystem_pthread.dylib             0x00007fff60c8d466 start_wqthread + 14
		// 27  ???                                 0xffffffffffffffff 0x0 + 18446744073709551615
		if tree.Address == "0xffffffffc" && len(tree.Children) == 1 {
			rootAct = newCallTreeP(tree.Children[0], symbolsByProfileID[tree.Children[0].ProfileID])
		} else {
			rootAct = newCallTreeP(tree, symbolsByProfileID[tree.ProfileID])
		}
		keys, err := agg.Update(rootAct, f.data.image, f.data.symbol)
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			threadNameToCounts, hasCounts := treeKeyToThreadCounts[key]
			uniqueprofileIDs, hasprofileIDs := treeKeyToprofileIDs[key]

			if !hasCounts {
				threadNameToCounts = make(map[string]uint64)
				treeKeyToThreadCounts[key] = threadNameToCounts
			}
			threadNameToCounts[tree.ThreadName] += 1

			if !hasprofileIDs {
				uniqueprofileIDs = make(map[string]bool)
				treeKeyToprofileIDs[key] = uniqueprofileIDs
			}
			uniqueprofileIDs[tree.ProfileID] = true
		}
	}

	// Convert call trees to the protocol buffer format
	callTrees := make([]CallTree, 0, len(agg.UniqueRootCallTrees))
	for key, tree := range agg.UniqueRootCallTrees {
		var profileIDs []string
		if uniqueprofileIDs, ok := treeKeyToprofileIDs[key]; ok {
			for profileID := range uniqueprofileIDs {
				profileIDs = append(profileIDs, profileID)
			}
		}
		var totalCount uint64
		if threadNameToCounts, ok := treeKeyToThreadCounts[key]; ok {
			for _, count := range threadNameToCounts {
				totalCount += count
			}
		}
		examplarprofileIDs := selectExemplarTraceIDs(profileIDs, defaultNExemplarTraces)
		sort.Strings(examplarprofileIDs)
		callTrees = append(callTrees, CallTree{
			ID:                key,
			Count:             totalCount,
			ThreadNameToCount: treeKeyToThreadCounts[key],
			ProfileIDs:        examplarprofileIDs,
			RootFrame:         newCallTreeFrameP(tree, nil, DisplayModeIOS),
		})
	}
	sortCallTrees(callTrees)

	return callTrees, nil
}

func sortCallTrees(pbs []CallTree) {
	sort.SliceStable(pbs, func(i, j int) bool {
		return pbs[i].ID < pbs[j].ID
	})
}

// newCallTree creates an `CallTree` from a `CallTree` by mapping
// the addresses in the call tree to the symbols from the batch symbolication
// response. This function does NOT perform recursive creation of the aggregate
// call trees, it only creates a single node.
func newCallTreeP(root *calltree.CallTreeP, symbols map[string]Symbol) (act *calltree.AggregateCallTree) {
	var totalDurationsNs, selfDurationsNs []float64
	rootDuration := calltree.TotalDurationP(root)
	if root.EndTimeNs != calltree.NoEndTime {
		totalDurationsNs = append(totalDurationsNs, float64(rootDuration))
	}
	if root.EndTimeNs != calltree.NoEndTime {
		selfDurationsNs = append(selfDurationsNs, float64(root.SelfTimeNs))
	}
	act = &calltree.AggregateCallTree{
		TotalDurationsNs: totalDurationsNs,
		SelfDurationsNs:  selfDurationsNs,
	}
	for _, child := range root.Children {
		act.Children = append(act.Children, newCallTreeP(child, symbols))
	}
	symbol, ok := symbols[root.Address]
	if !ok {
		return act
	}
	act.Image = symbol.Image
	act.Symbol = symbol.Name
	if line, path, ok := symbol.GetPath(); ok {
		act.Line = uint32(line)
		act.Path = path
	}
	return act
}

// selectExemplarTraceIDs selects and `n` trace IDs that are intended to represent
// exemplars of the overall set of traces. This logic currently just picks them
// randomly, but in the future we could use other heuristics.
func selectExemplarTraceIDs(ids []string, n int) []string {
	rIds := make([]string, 0, len(ids))
	perm := rand.Perm(len(ids))
	for _, i := range perm {
		rIds = append(rIds, ids[i])
	}
	return rIds[:min(len(ids), n)]
}

func accumulateFunctionCallsP(root, cur *calltree.CallTreeP, profileID string, functionCalls *[]functionCallP) {
	if cur.EndTimeNs == calltree.NoEndTime {
		for _, child := range cur.Children {
			accumulateFunctionCallsP(root, child, profileID, functionCalls)
		}
	} else {
		curDurationNs := calltree.TotalDurationP(cur)
		for _, child := range cur.Children {
			childDurationNs := calltree.TotalDurationP(child)
			if childDurationNs > curDurationNs {
				log.Error().
					Str("profile_id", profileID).
					Str("address", child.Address).
					Uint64("current_duration_ns", curDurationNs).
					Uint64("child_duration_ns", childDurationNs).
					Msg("child has longer duration than its parent")
				continue
			}
			accumulateFunctionCallsP(root, child, profileID, functionCalls)
		}
		*functionCalls = append(*functionCalls, functionCallP{
			address:      cur.Address,
			durationNs:   curDurationNs,
			isMainThread: root.IsMainThread,
			profileID:    profileID,
			rootCallTree: root,
			threadID:     cur.ThreadID,
			threadName:   root.ThreadName,
		})
	}
}
