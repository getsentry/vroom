package aggregate

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/getsentry/vroom/internal/android"
	"github.com/getsentry/vroom/internal/calltree"
	"github.com/getsentry/vroom/internal/errorutil"
	"github.com/getsentry/vroom/internal/quantile"
	"github.com/getsentry/vroom/internal/snubautil"
)

/*
	Android traces in their initial format consist of a list of thread, a list of methods, and a list of records
	corresponding to an entry or exit of a given method on a given thread.

	The are two main processing steps:
	1. Calls corresponding to the same method are aggregated for each trace.
	2. Method call aggregates are then themselves aggregated across all traces.
		a. First a quantile of the duration of each method is calculated in order to sort them and select the top ones.
        b. All necessary metrics are calculated for the top methods.
*/

type (
	AndroidTraceAggregatorP struct {
		individualProfiles     []individualProfile
		methodKeyToMethod      map[methodKey]android.AndroidMethod
		methodKeyToProfileData map[methodKey][]profileMethodData
		methodKeyToProfileIDs  map[methodKey][]string
		numFunctions           int
		profileCount           int
		profileIDToInteraction map[string]string
	}

	individualProfile struct {
		id                     string
		threadNames            map[threadIDType]string
		methodsInfoPerThreadID map[threadIDType][]methodInfoP
	}
)

func (a *AndroidTraceAggregatorP) SetTopNFunctions(n int) {
	a.numFunctions = n
}

func (aggregator *AndroidTraceAggregatorP) UpdateFromProfile(profile snubautil.Profile) error {
	var androidProfile android.AndroidProfile
	err := json.Unmarshal([]byte(profile.Profile), &androidProfile)
	if err != nil {
		return err
	}

	// Method IDs are only valid within their profiles whereas method keys are global.
	// To process records it's necessary to map method IDs to method keys.
	methodIDToKey := make(map[methodIDType]methodKey)
	for _, method := range androidProfile.Methods {
		mKey := keyP(method)

		// Only valid for the current trace.
		methodIDToKey[methodIDType(method.ID)] = mKey

		// Method information will be needed to convert to function calls.
		if _, found := aggregator.methodKeyToMethod[mKey]; !found {
			aggregator.methodKeyToMethod[mKey] = method
		}

		// Profile IDs for each method will be needed to convert to function calls.
		aggregator.methodKeyToProfileIDs[mKey] = append(aggregator.methodKeyToProfileIDs[mKey], profile.ProfileID)
	}

	// Thread IDs are only valid within their trace whereas thread names are global. To process records it's necessary
	// to map thread IDs to thread names. For convenience the main thread ID is also made available directly.
	var mainThreadID threadIDType
	threadIDToThreadName := make(map[threadIDType]string)
	for _, thread := range androidProfile.Threads {
		tID := threadIDType(thread.ID)
		if thread.Name == "main" {
			mainThreadID = tID
		}
		threadIDToThreadName[tID] = thread.Name
	}

	// Each method present in this profile is associated with one profileMethodData instance. The profileMethodData is mutated
	// as records are processed.
	methodKeyToProfileData := make(map[methodKey]profileMethodData)

	// Process records one by one.
	for i, event := range androidProfile.Events {
		methodID := methodIDType(event.MethodID)
		mKey := methodIDToKey[methodID]
		profileData, found := methodKeyToProfileData[mKey]
		if !found {
			profileData = newProfileMethodData()
		}
		err := profileData.update(mainThreadID, threadIDToThreadName, event, mKey, i)
		if err != nil {
			return err
		}
		methodKeyToProfileData[mKey] = profileData
	}

	it := individualProfile{
		id:                     profile.ProfileID,
		threadNames:            make(map[threadIDType]string),
		methodsInfoPerThreadID: make(map[threadIDType][]methodInfoP),
	}

	// Add the data corresponding to this trace to the aggregator.
	for methodKey, profileData := range methodKeyToProfileData {
		// Some methods never exit during a profile. For example ActivityThread.main or Looper.loop. For now we exclude
		// those methods from the results.
		if profileData.callExitCount == 0 {
			continue
		}

		for tID, methodsInfo := range profileData.methodsPerThreadID {
			it.threadNames[tID] = threadIDToThreadName[tID]
			it.methodsInfoPerThreadID[tID] = append(it.methodsInfoPerThreadID[tID], methodsInfo...)
		}

		aggregator.methodKeyToProfileData[methodKey] = append(aggregator.methodKeyToProfileData[methodKey], profileData)
	}

	aggregator.individualProfiles = append(aggregator.individualProfiles, it)
	aggregator.profileIDToInteraction[profile.ProfileID] = profile.TransactionName

	// Increment the trace count at the end. If there are errors the trace shouldn't be counted here.
	aggregator.profileCount++

	return nil
}

func (a *AndroidTraceAggregatorP) Result() (BacktraceAggregate, error) {

	// Only a subset of methods are usually returned, based on AndroidTraceAggregator.numFunctions. Duration is used to
	// sort and filter them, so it is calculated first.
	methodsSortedByDuration := a.methodsSortedByDuration()
	topMethods := make([]*methodWithDuration, 0, min(a.numFunctions, len(methodsSortedByDuration)))
	for _, mwd := range methodsSortedByDuration {
		// The name of certain methods can be unknown in some unusual cases. Records will still reference those methods
		// by ID, but since we don't know what they are it is not useful to process them or display them to users as top
		// functions.
		method, found := a.methodKeyToMethod[mwd.methodKey]
		if !found || method.Name == "" {
			continue
		}

		topMethods = append(topMethods, mwd)
		if len(topMethods) == a.numFunctions {
			break
		}
	}

	// Compute the rest of the metrics for the top methods only.
	functionCalls, err := a.functionCalls(topMethods)
	if err != nil {
		return BacktraceAggregate{}, err
	}

	methodsToCallTrees, err := a.methodsToCallTrees()
	if err != nil {
		return BacktraceAggregate{}, err
	}

	callTrees := make(map[string][]AggregateCallTree, len(topMethods))
	for _, topMethod := range topMethods {
		method := a.methodKeyToMethod[topMethod.methodKey]
		topMethodPackageName, topMethodSimpleMethodName, err := android.ExtractPackageNameAndSimpleMethodNameFromAndroidMethod(&method)
		if err != nil {
			continue
		}

		mctData, found := methodsToCallTrees[topMethodSimpleMethodName]
		if !found {
			continue
		}

		cta := calltree.NewCallTreeAggregator()
		treeKeyToProfileIDs := make(map[string]map[string]struct{})
		treeKeyToThreadNameToThreadCount := make(map[string]map[string]uint64)
		topMethodKey := computeMethodKey(topMethodPackageName, topMethodSimpleMethodName)

		for _, info := range mctData {
			keys, err := cta.Update(info.CallTree, topMethodPackageName, topMethodSimpleMethodName)
			if err != nil {
				return BacktraceAggregate{}, err
			}

			for _, key := range keys {
				threadNameToThreadCount, countFound := treeKeyToThreadNameToThreadCount[key]
				if !countFound {
					threadNameToThreadCount = make(map[string]uint64)
					treeKeyToThreadNameToThreadCount[key] = threadNameToThreadCount
				}
				threadNameToThreadCount[info.ThreadName] += 1

				uniqueProfileIDs, profileIDsFound := treeKeyToProfileIDs[key]
				if !profileIDsFound {
					uniqueProfileIDs = make(map[string]struct{})
					treeKeyToProfileIDs[key] = uniqueProfileIDs
				}
				uniqueProfileIDs[info.ProfileID] = struct{}{}
			}
		}

		callTrees[topMethodKey] = make([]AggregateCallTree, 0, len(cta.UniqueRootCallTrees))
		for id, tree := range cta.UniqueRootCallTrees {
			var profileIDs []string
			for profileID, _ := range treeKeyToProfileIDs[id] {
				profileIDs = append(profileIDs, profileID)
			}

			var totalCount uint64
			for _, count := range treeKeyToThreadNameToThreadCount[id] {
				totalCount += count
			}

			callTrees[topMethodKey] = append(callTrees[topMethodKey], AggregateCallTree{
				Count:             totalCount,
				ID:                id,
				RootFrame:         newCallTreeFrameP(tree, nil, DisplayModeAndroid),
				ThreadNameToCount: treeKeyToThreadNameToThreadCount[id],
				ProfileIDs:        profileIDs,
			})
		}
	}

	functionToCallTrees := make(map[string][]AggregateCallTree)
	for topMethodKey, aggCallTrees := range callTrees {
		sortAggregateCallTrees(aggCallTrees)
		functionToCallTrees[topMethodKey] = aggCallTrees
	}

	return BacktraceAggregate{
		FunctionCalls:       functionCalls,
		FunctionToCallTrees: functionToCallTrees,
	}, nil

}

type CallTreeInfo struct {
	CallTree   *calltree.AggregateCallTree
	ThreadName string
	TraceID    string
}

type CallTreeInfoP struct {
	CallTree   *calltree.AggregateCallTree
	ThreadName string
	ProfileID  string
}

func (a *AndroidTraceAggregatorP) methodsToCallTrees() (map[string][]CallTreeInfoP, error) {
	callTrees := make(map[string][]CallTreeInfoP)
	for _, profile := range a.individualProfiles {
		for threadID, methodsInfo := range profile.methodsInfoPerThreadID {
			sort.SliceStable(methodsInfo, func(i, j int) bool {
				// Order according to the record index extracted from the trace
				// Since the index is captured on the EXIT action, a higher value means the function was called first
				return methodsInfo[j].index < methodsInfo[i].index
			})

			root := &node{
				relativeEndUsec: math.MaxUint32,
				calltree: &calltree.AggregateCallTree{
					Symbol: "root",
				},
			}

			for _, mi := range methodsInfo {
				method := a.methodKeyToMethod[mi.mk]

				var packageName, simpleMethodName string
				if method.Signature == "" {
					packageName = method.ClassName
					simpleMethodName = method.Name
				} else {
					var err error
					packageName, simpleMethodName, err = android.ExtractPackageNameAndSimpleMethodNameFromAndroidMethod(&method)
					if err != nil {
						return nil, err
					}
				}

				ct := &calltree.AggregateCallTree{
					Image:  packageName,
					Symbol: simpleMethodName,
					Line:   uint32(method.SourceLine),
					Path:   method.SourceFile,
				}
				n := &node{
					relativeStartUsec: uint32(mi.relativeStartNs),
					relativeEndUsec:   uint32(mi.relativeEndNs),
					calltree:          ct,
				}

				root.Insert(n)
			}

			// Compute durations for all the nodes
			root.ComputeDurationsNs()

			threadName := profile.threadNames[threadID]
			for _, c := range root.children {
				for _, m := range c.calltree.Symbols() {
					callTrees[m] = append(callTrees[m], CallTreeInfoP{
						CallTree:   c.calltree,
						ThreadName: threadName,
						ProfileID:  profile.id,
					})
				}
			}
		}
	}

	return callTrees, nil
}

type node struct {
	relativeStartUsec uint32
	relativeEndUsec   uint32
	children          []*node
	calltree          *calltree.AggregateCallTree
}

func (n *node) DurationNs() float64 {
	return float64(n.relativeEndUsec-n.relativeStartUsec) * 1000
}

func (n *node) Wraps(v *node) bool {
	return n.relativeStartUsec <= v.relativeStartUsec && v.relativeEndUsec <= n.relativeEndUsec
}

func (n *node) Insert(v *node) {
	for _, c := range n.children {
		if c.Wraps(v) {
			c.Insert(v)
			return
		}
	}

	n.calltree.Children = append(n.calltree.Children, v.calltree)
	n.children = append(n.children, v)
}

func (n *node) ComputeDurationsNs() {
	var childrenDurationNs float64

	for _, c := range n.children {
		c.ComputeDurationsNs()
		childrenDurationNs += c.DurationNs()
	}

	durationNs := n.DurationNs()
	selfDurationNs := float64(durationNs - childrenDurationNs)

	n.calltree.TotalDurationsNs = append(n.calltree.TotalDurationsNs, durationNs)
	n.calltree.SelfDurationsNs = append(n.calltree.SelfDurationsNs, selfDurationNs)
}

// Should only be called after methodKeyToProfileData has been populated.
func (aggregator *AndroidTraceAggregatorP) methodsSortedByDuration() []*methodWithDuration {
	var methodsWithDuration []*methodWithDuration
	for mk, profileDataList := range aggregator.methodKeyToProfileData {
		methodWithDuration := &methodWithDuration{methodKey: mk}

		for _, profileData := range profileDataList {
			methodWithDuration.durationsNs.Add(profileData.durationsNs...)
		}

		methodWithDuration.durationNsP75 = methodWithDuration.durationsNs.Percentile(0.75)
		methodsWithDuration = append(methodsWithDuration, methodWithDuration)
	}

	sort.SliceStable(methodsWithDuration, func(i, j int) bool {
		// Sort by inverse duration, longest first
		if methodsWithDuration[i].durationNsP75 != methodsWithDuration[j].durationNsP75 {
			if math.IsNaN(float64(methodsWithDuration[i].durationNsP75)) {
				return false
			}
			if math.IsNaN(float64(methodsWithDuration[j].durationNsP75)) {
				return true
			}
			return methodsWithDuration[j].durationNsP75 < methodsWithDuration[i].durationNsP75
		}
		return methodsWithDuration[i].methodKey.Less(methodsWithDuration[j].methodKey)
	})

	return methodsWithDuration
}

func (a *AndroidTraceAggregatorP) computeFunctionCall(mwd *methodWithDuration) (BacktraceAggregateFunctionCall, error) {
	mk := mwd.methodKey
	method, found := a.methodKeyToMethod[mk]
	if !found {
		return BacktraceAggregateFunctionCall{}, fmt.Errorf("androidtrace: %w: did not find AndroidProfile method corresponding to method %v", errorutil.ErrDataIntegrity, mk)
	}

	packageName, simpleMethodName, err := android.ExtractPackageNameAndSimpleMethodNameFromAndroidMethod(&method)
	if err != nil {
		return BacktraceAggregateFunctionCall{}, err
	}

	traceDataList, found := a.methodKeyToProfileData[mk]
	if !found {
		return BacktraceAggregateFunctionCall{}, fmt.Errorf("androidtrace: %w: did not find profile data list corresponding to method %v", errorutil.ErrDataIntegrity, mk)
	}

	// Number of profiles that contained this method.
	methodProfileCount := len(traceDataList)
	if methodProfileCount > a.profileCount {
		return BacktraceAggregateFunctionCall{}, fmt.Errorf("androidtrace: %w: the number of profiles associated with one method cannot be greater than the total number of traces", errorutil.ErrDataIntegrity)
	}

	var frequency []float64
	var callCount float32

	mainThreadCallCount := 0
	threadNameToCallCount := make(map[string]int)

	for _, traceData := range traceDataList {
		// For each trace where the method wasn't present a frequency value of 0 is added.
		values := make([]float64, a.profileCount-methodProfileCount+1)
		values[0] = float64(traceData.callExitCount)
		frequency = append(frequency, values...)

		callCount += float32(traceData.callExitCount)
		mainThreadCallCount += traceData.mainThreadCallExitCount
		for threadName, callCount := range traceData.threadNameToCallExitCount {
			threadNameToCallCount[threadName] += callCount
		}
	}

	threadNameToPercent := make(map[string]float32)
	for threadName, threadCallCount := range threadNameToCallCount {
		threadNameToPercent[threadName] = float32(threadCallCount) / callCount
	}

	profileIDs, found := a.methodKeyToProfileIDs[mk]
	if !found {
		return BacktraceAggregateFunctionCall{}, fmt.Errorf("androidtrace: %w: did not find profile IDs corresponding to method %v", errorutil.ErrDataIntegrity, mk)
	}

	var interactions []string
	uniqueInteractions := make(map[string]struct{})
	for _, t := range profileIDs {
		i, exists := a.profileIDToInteraction[t]
		if !exists {
			continue
		}
		if _, exists := uniqueInteractions[i]; exists {
			continue
		}
		interactions = append(interactions, i)
		uniqueInteractions[i] = struct{}{}
	}

	return BacktraceAggregateFunctionCall{
		Key:                 computeMethodKey(packageName, simpleMethodName),
		Image:               packageName,
		Symbol:              simpleMethodName,
		DurationNs:          quantileToAggQuantiles(mwd.durationsNs),
		Frequency:           quantileToAggQuantiles(quantile.Quantile{Xs: frequency}),
		MainThreadPercent:   float32(mainThreadCallCount) / callCount,
		ThreadNameToPercent: threadNameToPercent,
		Line:                int(method.SourceLine),
		Path:                method.SourceFile,
		ProfileIDs:          profileIDs,
		TransactionNames:    interactions,
	}, nil
}

func computeMethodKey(packageName, simpleMethodName string) string {
	h := computeFunctionHash(packageName, simpleMethodName)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Should only be called after methodKeyToMethod, methodKeyToProfileIDs, and methodKeyToProfileData have been populated.
func (aggregator *AndroidTraceAggregatorP) functionCalls(methodsWithDuration []*methodWithDuration) ([]BacktraceAggregateFunctionCall, error) {
	functionCalls := make([]BacktraceAggregateFunctionCall, 0, len(methodsWithDuration))

	for _, methodWithDuration := range methodsWithDuration {
		mwdProto, err := aggregator.computeFunctionCall(methodWithDuration)

		if err != nil {
			continue
		}

		functionCalls = append(functionCalls, mwdProto)
	}

	return functionCalls, nil
}

type (
	threadIDType uint32
	methodIDType uint32

	methodKey struct {
		className string
		name      string
		signature string
	}

	methodInfoP struct {
		index           int
		mk              methodKey
		relativeStartNs uint64
		relativeEndNs   uint64
	}

	// Data about a given method from a single trace.
	profileMethodData struct {
		// Used for processing.
		enterRecords       map[threadIDType][]AndroidtraceRecord
		methodsPerThreadID map[threadIDType][]methodInfoP

		// Results after processing is done.
		callExitCount             int
		mainThreadCallExitCount   int
		threadNameToCallExitCount map[string]int
		durationsNs               []float64
	}

	AndroidtraceRecord struct {
		// Whether we are entering or exiting the method.
		MethodAction string

		// The thread id where the method is running
		ThreadID uint32
		// The method id
		MethodID uint32
		/**
		 * The CPU time delta since the start of the transaction, in nanoseconds. Optional (some devices
		 * may not record it) but guaranteed to be set if wall_time_since_start_ns isn't.
		 */
		TimeDeltaSinceStartNs uint64
		/**
		 * The wall time since the start of the trace, in nanoseconds. Optional (some devices may not
		 * record it) but guaranteed to be set if time_delta_since_start_ns isn't.
		 */
		WallTimeSinceStartNs uint64
	}

	// An intermediary struct to compute the duration of methods before the rest.
	methodWithDuration struct {
		methodKey     methodKey
		durationsNs   quantile.Quantile
		durationNsP75 float64
	}
)

func (mk methodKey) Less(other methodKey) bool {
	if mk.className != other.className {
		return mk.className < other.className
	}

	if mk.name != other.name {
		return mk.name < other.name
	}

	return mk.signature < other.signature
}

func newProfileMethodData() profileMethodData {
	return profileMethodData{
		enterRecords:              make(map[threadIDType][]AndroidtraceRecord),
		threadNameToCallExitCount: make(map[string]int),
		methodsPerThreadID:        make(map[threadIDType][]methodInfoP),
	}
}

func keyP(method android.AndroidMethod) methodKey {
	return methodKey{
		className: method.ClassName,
		name:      method.Name,
		signature: method.Signature,
	}
}

// Profile events must be processed in order.
func (data *profileMethodData) update(mainThreadID threadIDType, threadIDToThreadName map[threadIDType]string, event android.AndroidEvent, mk methodKey, index int) error {
	threadID := threadIDType(event.ThreadID)

	switch event.Action {
	case "Enter":
		data.enterRecords[threadID] = append(data.enterRecords[threadID], eventToAndroidTraceRecord(event))
	case "Exit":
		enterEvents := data.enterRecords[threadID]
		if len(enterEvents) == 0 {
			return fmt.Errorf("androidtrace: %w: did not find an enter event for exit record %v", errorutil.ErrDataIntegrity, event)
		}

		lastIndex := len(enterEvents) - 1
		enterEvent := enterEvents[lastIndex]
		// Remove the last (most recent) enter record now that the corresponding exit is being processed.
		data.enterRecords[threadID] = enterEvents[:lastIndex]

		// Counts are incremented on exit so that calls whose exit is not recorded, or who end due to an exception, are
		// not included.
		data.callExitCount++
		if threadID == mainThreadID {
			data.mainThreadCallExitCount++
		}
		threadName, found := threadIDToThreadName[threadID]
		if !found {
			threadName = strconv.FormatUint(event.ThreadID, 10)
		}
		data.threadNameToCallExitCount[threadName]++

		enterRecordedAt := RecordedAt(enterEvent)
		currentRecordedAt := RecordedAt(eventToAndroidTraceRecord(event))
		data.methodsPerThreadID[threadID] = append(data.methodsPerThreadID[threadID], methodInfoP{
			relativeStartNs: enterRecordedAt,
			relativeEndNs:   currentRecordedAt,
			mk:              mk,
			index:           index,
		})
		durationNs := float64((currentRecordedAt - enterRecordedAt))
		data.durationsNs = append(data.durationsNs, durationNs)
	case "Unwind":
		if enterEvents, exists := data.enterRecords[threadID]; exists {
			lastIndex := len(enterEvents) - 1
			enterEvent := enterEvents[lastIndex]

			// Still count the record for the thread
			data.methodsPerThreadID[threadID] = append(data.methodsPerThreadID[threadID], methodInfoP{
				relativeStartNs: RecordedAt(enterEvent),
				relativeEndNs:   RecordedAt(eventToAndroidTraceRecord(event)),
				mk:              mk,
				index:           index,
			})

			// Remove the last (most recent) record.
			data.enterRecords[threadID] = enterEvents[:lastIndex]
		}
	}

	return nil
}

func RecordedAt(r AndroidtraceRecord) uint64 {
	if r.WallTimeSinceStartNs > 0 {
		return r.WallTimeSinceStartNs
	}
	return r.TimeDeltaSinceStartNs
}

func eventToAndroidTraceRecord(event android.AndroidEvent) AndroidtraceRecord {
	return AndroidtraceRecord{
		MethodAction:          event.Action,
		ThreadID:              uint32(event.ThreadID),
		MethodID:              uint32(event.MethodID),
		TimeDeltaSinceStartNs: event.Time.Monotonic.Cpu.Nanos,
		WallTimeSinceStartNs:  event.Time.Monotonic.Wall.Nanos,
	}
}
