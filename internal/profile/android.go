package profile

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"path"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/android"
	"github.com/getsentry/vroom/internal/errorutil"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/packageutil"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/speedscope"
)

type (
	AndroidThread struct {
		ID   uint64 `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	}

	AndroidMethod struct {
		ClassName    string            `json:"class_name,omitempty"`
		Data         Data              `json:"data"`
		ID           uint64            `json:"id,omitempty"`
		InlineFrames []AndroidMethod   `json:"inline_frames,omitempty"`
		Name         string            `json:"name,omitempty"`
		Signature    string            `json:"signature,omitempty"`
		SourceFile   string            `json:"source_file,omitempty"`
		SourceLine   uint32            `json:"source_line,omitempty"`
		SourceCol    uint32            `json:"-"`
		InApp        *bool             `json:"in_app"`
		Platform     platform.Platform `json:"platform,omitempty"`
	}

	Data struct {
		DeobfuscationStatus string `json:"deobfuscation_status,omitempty"`
		// for react-native apps where we have js frames turned into android methods
		JsSymbolicated *bool `json:"symbolicated,omitempty"`
		OrigInApp      *int8 `json:"orig_in_app,omitempty"`
	}
)

func (m AndroidMethod) isApplicationFrame(appIdentifier string) bool {
	if appIdentifier != "" {
		return strings.HasPrefix(m.ClassName, appIdentifier+".")
	}
	return packageutil.IsAndroidApplicationPackage(m.ClassName)
}

func (m AndroidMethod) Frame() frame.Frame {
	className, _, err := m.ExtractPackageNameAndSimpleMethodNameFromAndroidMethod()
	if err != nil {
		className = m.ClassName
	}
	methodName, err := m.FullMethodNameFromAndroidMethod()
	if err != nil {
		methodName = m.Name
	}
	var inApp bool
	if m.InApp != nil {
		inApp = *m.InApp
	} else {
		inApp = packageutil.IsAndroidApplicationPackage(m.ClassName)
	}
	return frame.Frame{
		Data: frame.Data{
			DeobfuscationStatus: m.Data.DeobfuscationStatus,
			JsSymbolicated:      m.Data.JsSymbolicated,
		},
		File:     path.Base(m.SourceFile),
		Function: methodName,
		InApp:    &inApp,
		Line:     m.SourceLine,
		Column:   m.SourceCol,
		MethodID: m.ID,
		Package:  className,
		Path:     m.SourceFile,
		Platform: m.Platform,
	}
}

func (m AndroidMethod) ExtractPackageNameAndSimpleMethodNameFromAndroidMethod() (string, string, error) {
	fullMethodName, err := m.FullMethodNameFromAndroidMethod()
	if err != nil {
		return "", "", err
	}

	packageName := m.packageNameFromAndroidMethod()

	return packageName, android.StripPackageNameFromFullMethodName(fullMethodName, packageName), nil
}

func (m AndroidMethod) FullMethodNameFromAndroidMethod() (string, error) {
	// when we we're dealing with js frame that were "converted"
	// to android methods (react-native) we don't have class name
	if m.ClassName == "" {
		return m.Name, nil
	}
	var builder strings.Builder
	builder.WriteString(m.ClassName)
	// "<init>" refers to the constructor in which case it's more readable to omit the method name. Note the method name
	// can also be a static initializer "<clinit>" but I don't know of any better ways to represent it so leaving as is.
	if m.Name != "<init>" {
		builder.WriteRune('.')
		builder.WriteString(m.Name)
	}
	builder.WriteString(m.Signature)

	return builder.String(), nil
}

func (m AndroidMethod) packageNameFromAndroidMethod() string {
	index := strings.LastIndex(m.ClassName, ".")

	if index == -1 {
		return m.ClassName
	}

	return m.ClassName[:index]
}

type EventMonotonic struct {
	Wall Duration `json:"wall,omitempty"`
	CPU  Duration `json:"cpu,omitempty"`
}

type EventTime struct {
	Global    Duration       `json:"global,omitempty"`
	Monotonic EventMonotonic `json:"Monotonic,omitempty"`
}

type Duration struct {
	Secs  uint64 `json:"secs,omitempty"`
	Nanos uint64 `json:"nanos,omitempty"`
}

type Action string

const (
	EnterAction  = "Enter"
	ExitAction   = "Exit"
	UnwindAction = "Unwind"
)

type AndroidEvent struct {
	Action   Action    `json:"action,omitempty"`
	ThreadID uint64    `json:"thread_id,omitempty"`
	MethodID uint64    `json:"method_id,omitempty"`
	Time     EventTime `json:"time,omitempty"`
}

type (
	Android struct {
		Clock     Clock           `json:"clock"`
		Events    []AndroidEvent  `json:"events,omitempty"`
		Methods   []AndroidMethod `json:"methods,omitempty"`
		StartTime uint64          `json:"start_time,omitempty"`
		// SdkStartTime, if set (manually), it's an absolute ts in Ns
		// whose value comes from the chunk timestamp set by the sentry SDK.
		// This is used to control the ts during callTree generation.
		SdkStartTime uint64          `json:"-"`
		Threads      []AndroidThread `json:"threads,omitempty"`
	}

	Clock string
)

const (
	DualClock   Clock = "Dual"
	CPUClock    Clock = "Cpu"
	WallClock   Clock = "Wall"
	GlobalClock Clock = "Global"

	mainThread = "main"
)

func (p Android) TimestampGetter() func(EventTime) uint64 {
	var buildTimestamp func(t EventTime) uint64
	switch p.Clock {
	case GlobalClock:
		buildTimestamp = func(t EventTime) uint64 {
			return t.Global.Secs*uint64(time.Second) + t.Global.Nanos - p.StartTime
		}
	case CPUClock:
		buildTimestamp = func(t EventTime) uint64 {
			return t.Monotonic.CPU.Secs*uint64(time.Second) + t.Monotonic.CPU.Nanos
		}
	default:
		buildTimestamp = func(t EventTime) uint64 {
			return t.Monotonic.Wall.Secs*uint64(time.Second) + t.Monotonic.Wall.Nanos
		}
	}
	return buildTimestamp
}

// maxTimeNs: the highest time (in nanoseconds) in the sequence so far
// latestNs: the latest time value in ns (at time t-1) before it was updated
// currentNs: current value in ns (at time t) before it's updated.
func getAdjustedTime(maxTimeNs, latestNs, currentNs uint64) uint64 {
	if currentNs < maxTimeNs && currentNs < latestNs {
		return maxTimeNs + 1e9
	}
	return maxTimeNs + (currentNs - latestNs)
}

// Wall-clock time is supposed to be monotonic
// in a few rare cases we've noticed this was not the case.
// Due to some overflow happening client-side in the embedded
// profiler, the sequence might be decreasing at certain points.
//
// This is just a workaround to mitigate this issue, should it
// happen.
func (p *Android) FixSamplesTime() {
	if p.Clock == GlobalClock || p.Clock == CPUClock {
		return
	}
	threadMaxTimeNs := make(map[uint64]uint64)
	threadLatestSampleTimeNs := make(map[uint64]uint64)
	regressionIndex := -1

	for i, event := range p.Events {
		current := (event.Time.Monotonic.Wall.Secs * 1e9) + event.Time.Monotonic.Wall.Nanos
		if current < threadLatestSampleTimeNs[event.ThreadID] {
			regressionIndex = i
			break
		}
		threadLatestSampleTimeNs[event.ThreadID] = current
		threadMaxTimeNs[event.ThreadID] = max(threadMaxTimeNs[event.ThreadID], current)
	}

	if regressionIndex > 0 {
		for i := regressionIndex; i < len(p.Events); i++ {
			event := p.Events[i]
			current := (event.Time.Monotonic.Wall.Secs * 1e9) + event.Time.Monotonic.Wall.Nanos

			newTime := getAdjustedTime(threadMaxTimeNs[event.ThreadID], threadLatestSampleTimeNs[event.ThreadID], current)
			threadMaxTimeNs[event.ThreadID] = max(threadMaxTimeNs[event.ThreadID], newTime)

			threadLatestSampleTimeNs[event.ThreadID] = current
			p.Events[i].Time.Monotonic.Wall.Secs = (newTime / 1e9)
			p.Events[i].Time.Monotonic.Wall.Nanos = (newTime % 1e9)
		}
	}
}

func (p *Android) AddTimeDelta(deltaNS int64) func(*AndroidEvent) error {
	var addDeltaTimestamp func(e *AndroidEvent) error
	timestampBuilder := p.TimestampGetter()
	switch p.Clock {
	case GlobalClock:
		addDeltaTimestamp = func(e *AndroidEvent) error {
			ts := timestampBuilder(e.Time)
			ts, err := getTsFromDelta(ts, deltaNS)
			if err != nil {
				return err
			}
			secs := (ts / 1e9)
			nanos := (ts % 1e9)
			e.Time.Global.Secs = secs
			e.Time.Global.Nanos = nanos
			return nil
		}
	case CPUClock:
		addDeltaTimestamp = func(e *AndroidEvent) error {
			ts := timestampBuilder(e.Time)
			ts, err := getTsFromDelta(ts, deltaNS)
			if err != nil {
				return err
			}
			secs := (ts / 1e9)
			nanos := (ts % 1e9)
			e.Time.Monotonic.CPU.Secs = secs
			e.Time.Monotonic.CPU.Nanos = nanos
			return nil
		}
	default:
		addDeltaTimestamp = func(e *AndroidEvent) error {
			ts := timestampBuilder(e.Time)
			ts, err := getTsFromDelta(ts, deltaNS)
			if err != nil {
				return err
			}
			secs := (ts / 1e9)
			nanos := (ts % 1e9)
			e.Time.Monotonic.Wall.Secs = secs
			e.Time.Monotonic.Wall.Nanos = nanos
			return nil
		}
	}
	return addDeltaTimestamp
}

func getTsFromDelta(ts uint64, deltaNS int64) (uint64, error) {
	if deltaNS < 0 && uint64(-deltaNS) <= ts {
		return ts - uint64(-deltaNS), nil
	} else if deltaNS >= 0 {
		return ts + uint64(deltaNS), nil
	}
	return 0, errors.New("error: cannot subtract a delta bigger than the timestamp itself")
}

// CallTrees generates call trees for a given profile.
func (p Android) CallTrees() map[uint64][]*nodetree.Node {
	return p.CallTreesWithMaxDepth(MaxStackDepth)
}

func (p Android) CallTreesWithMaxDepth(maxDepth int) map[uint64][]*nodetree.Node {
	// in case wall-clock.secs is not monotonic, "fix" it
	p.FixSamplesTime()

	var activeThreadID uint64
	for _, thread := range p.Threads {
		if thread.Name == mainThread {
			activeThreadID = thread.ID
			break
		}
	}

	buildTimestamp := p.TimestampGetter()
	treesByThreadID := make(map[uint64][]*nodetree.Node)
	stacks := make(map[uint64][]*nodetree.Node)
	stackDepth := make(map[uint64]int)

	methods := make(map[uint64]AndroidMethod)
	for _, m := range p.Methods {
		methods[m.ID] = m
	}

	closeFrame := func(threadID uint64, ts uint64, i int) {
		n := stacks[threadID][i]
		n.Update(ts)
		n.SampleCount = int(math.Ceil(float64(n.DurationNS) / float64((10 * time.Millisecond))))
	}

	var maxTimestampNS uint64
	enterPerMethod := make(map[uint64]int)
	exitPerMethod := make(map[uint64]int)

	for _, e := range p.Events {
		if e.ThreadID != activeThreadID {
			continue
		}

		ts := buildTimestamp(e.Time) + p.SdkStartTime
		if ts > maxTimestampNS {
			maxTimestampNS = ts
		}

		switch e.Action {
		case EnterAction:
			m, exists := methods[e.MethodID]
			if !exists {
				methods[e.MethodID] = AndroidMethod{
					ClassName: "unknown",
					ID:        e.MethodID,
					Name:      "unknown",
				}
			}
			stackDepth[e.ThreadID]++
			if stackDepth[e.ThreadID] > maxDepth {
				continue
			}
			enterPerMethod[e.MethodID]++
			n := nodetree.NodeFromFrame(m.Frame(), ts, 0, 0)
			if len(stacks[e.ThreadID]) == 0 {
				treesByThreadID[e.ThreadID] = append(treesByThreadID[e.ThreadID], n)
			} else {
				i := len(stacks[e.ThreadID]) - 1
				stacks[e.ThreadID][i].Children = append(stacks[e.ThreadID][i].Children, n)
			}
			stacks[e.ThreadID] = append(stacks[e.ThreadID], n)
			n.Fingerprint = generateFingerprint(stacks[e.ThreadID])
		case ExitAction, UnwindAction:
			stackDepth[e.ThreadID]--
			if stackDepth[e.ThreadID] > maxDepth {
				continue
			}
			if len(stacks[e.ThreadID]) == 0 {
				continue
			}
			i := len(stacks[e.ThreadID]) - 1
			var eventSkipped bool
			for ; i >= 0; i-- {
				n := stacks[e.ThreadID][i]
				if n.Frame.MethodID != e.MethodID &&
					enterPerMethod[e.MethodID] <= exitPerMethod[e.MethodID] {
					eventSkipped = true
					break
				}
				closeFrame(e.ThreadID, ts, i)
				exitPerMethod[e.MethodID]++
				if n.Frame.MethodID == e.MethodID {
					break
				}
			}
			// If we didn't skip the event, we should cut the stack accordingly.
			if !eventSkipped {
				stacks[e.ThreadID] = stacks[e.ThreadID][:i]
			}
		}
	}
	// Close remaining open frames.
	for threadID, stack := range stacks {
		for i := len(stack) - 1; i >= 0; i-- {
			closeFrame(threadID, maxTimestampNS, i)
		}
	}
	for _, trees := range treesByThreadID {
		for _, root := range trees {
			root.Close(maxTimestampNS)
		}
	}
	return treesByThreadID
}

func generateFingerprint(stack []*nodetree.Node) uint64 {
	h := fnv.New64()
	for _, n := range stack {
		n.WriteToHash(h)
	}
	return h.Sum64()
}

func (p *Android) NormalizeMethods(pi profileInterface) {
	metadata := pi.GetTransactionMetadata()
	appIdentifier := metadata.AppIdentifier

	for i := range p.Methods {
		method := p.Methods[i]

		for j := range method.InlineFrames {
			inlineMethod := method.InlineFrames[j]
			if inlineMethod.Data.OrigInApp != nil {
				continue
			}

			inApp := inlineMethod.isApplicationFrame(appIdentifier)
			inlineMethod.InApp = &inApp

			method.InlineFrames[j] = inlineMethod
		}
		// If a stack trace rule was applied to a given
		// frame this should have the precedence over the
		// appIdentifier.
		if method.Data.OrigInApp != nil {
			continue
		}

		inApp := method.isApplicationFrame(appIdentifier)
		method.InApp = &inApp

		p.Methods[i] = method
	}
}

func (p Android) Speedscope() (speedscope.Output, error) {
	return p.SpeedscopeWithMaxDepth(MaxStackDepth)
}

func (p Android) SpeedscopeWithMaxDepth(maxDepth int) (speedscope.Output, error) {
	// in case wall-clock.secs is not monotonic, "fix" it
	p.FixSamplesTime()

	frames := make([]speedscope.Frame, 0)
	methodIDToFrameIndex := make(map[uint64][]int)
	for _, method := range p.Methods {
		if len(method.InlineFrames) > 0 {
			for _, m := range method.InlineFrames {
				var inApp bool
				if m.InApp != nil {
					inApp = *m.InApp
				} else {
					inApp = packageutil.IsAndroidApplicationPackage(m.ClassName)
				}
				methodIDToFrameIndex[method.ID] = append(
					methodIDToFrameIndex[method.ID],
					len(frames),
				)
				frames = append(frames, speedscope.Frame{
					File:          m.SourceFile,
					Image:         m.ClassName,
					Inline:        true,
					IsApplication: inApp,
					Line:          m.SourceLine,
					Col:           m.SourceCol,
					Name:          m.Name,
				})
			}
		} else {
			packageName, _, err := method.ExtractPackageNameAndSimpleMethodNameFromAndroidMethod()
			if err != nil {
				return speedscope.Output{}, err
			}
			fullMethodName, err := method.FullMethodNameFromAndroidMethod()
			if err != nil {
				return speedscope.Output{}, err
			}
			var inApp bool
			if method.InApp != nil {
				inApp = *method.InApp
			} else {
				inApp = packageutil.IsAndroidApplicationPackage(method.ClassName)
			}
			methodIDToFrameIndex[method.ID] = append(methodIDToFrameIndex[method.ID], len(frames))
			frames = append(frames, speedscope.Frame{
				Name:          fullMethodName,
				File:          method.SourceFile,
				Line:          method.SourceLine,
				Col:           method.SourceCol,
				IsApplication: inApp,
				Image:         packageName,
			})
		}
	}

	emitEvent := func(p *speedscope.EventedProfile, et speedscope.EventType, methodID, ts uint64) {
		frameIndexes, ok := methodIDToFrameIndex[methodID]
		if !ok {
			// sometimes it might happen that a method is listed in events but an entry definition
			// is not correctly defined in the methods entry. We don't wan't to fail the whole chrometrace
			// for this so we create a method on the fly
			frameIndexes = []int{len(frames)}
			methodIDToFrameIndex[methodID] = append(methodIDToFrameIndex[methodID], frameIndexes[0])
			frames = append(frames, speedscope.Frame{
				Name:          fmt.Sprintf("unknown (id %d)", methodID),
				File:          "unknown",
				Line:          0,
				IsApplication: false,
				Image:         "unknown",
			})
		}
		for _, fi := range frameIndexes {
			p.Events = append(p.Events, speedscope.Event{
				Type:  et,
				Frame: fi,
				At:    ts,
			})
		}
	}

	threadIDToProfile := make(map[uint64]*speedscope.EventedProfile)
	methodStacks := make(map[uint64][]uint64) // map of thread ID -> stack of method IDs
	stackDepth := make(map[uint64]int)
	buildTimestamp := p.TimestampGetter()

	enterPerMethod := make(map[uint64]map[uint64]int)
	exitPerMethod := make(map[uint64]map[uint64]int)

	for _, event := range p.Events {
		ts := buildTimestamp(event.Time)
		prof, ok := threadIDToProfile[event.ThreadID]
		if !ok {
			threadID := event.ThreadID
			prof = &speedscope.EventedProfile{
				StartValue: ts,
				ThreadID:   threadID,
				Type:       speedscope.ProfileTypeEvented,
				Unit:       speedscope.ValueUnitNanoseconds,
			}
			threadIDToProfile[threadID] = prof
		}
		prof.EndValue = ts

		switch event.Action {
		case "Enter":
			stackDepth[event.ThreadID]++
			if stackDepth[event.ThreadID] > maxDepth {
				continue
			}
			if _, ok := enterPerMethod[event.ThreadID]; !ok {
				enterPerMethod[event.ThreadID] = make(map[uint64]int)
			}
			enterPerMethod[event.ThreadID][event.MethodID]++
			methodStacks[event.ThreadID] = append(methodStacks[event.ThreadID], event.MethodID)
			emitEvent(prof, speedscope.EventTypeOpenFrame, event.MethodID, ts)
		case "Exit", "Unwind":
			stackDepth[event.ThreadID]--
			if stackDepth[event.ThreadID] > maxDepth {
				continue
			}
			stack := methodStacks[event.ThreadID]
			if len(stack) == 0 {
				// This case happens when we filter events for a given transaction.
				// The enter event might be started before the transaction but finishes during.
				// In this case, we choose to ignore it.
				continue
			}
			i := len(stack) - 1
			// Iterate from top -> bottom of stack, looking for the method we're attempting to end.
			// Typically, this method should be on the top of the stack, but we may also be trying to
			// end a method before explicitly ending the child methods that are on top of that method
			// in the stack. In this scenario, we will synthesize end events for all methods that have
			// not been explicitly ended, matching the behavior of the Chrome trace viewer. Speedscope
			// handles this scenario a different way by doing nothing and leaving these methods with
			// indefinite durations.
			var eventSkipped bool
			for ; i >= 0; i-- {
				methodID := stack[i]
				// Skip exit event when we didn't record an enter event for that method.
				if methodID != event.MethodID && enterPerMethod[event.ThreadID][event.MethodID] <= exitPerMethod[event.ThreadID][event.MethodID] {
					eventSkipped = true
					break
				}
				emitEvent(prof, speedscope.EventTypeCloseFrame, methodID, ts)
				if _, ok := exitPerMethod[event.ThreadID]; !ok {
					exitPerMethod[event.ThreadID] = make(map[uint64]int)
				}
				exitPerMethod[event.ThreadID][methodID]++
				// Pop the elements that we emitted end events for off the stack
				// Keep closing methods until we closed the one we intended to close
				if methodID == event.MethodID {
					break
				}
			}
			// If we didn't skip the event, we should cut the stack accordingly.
			if !eventSkipped {
				methodStacks[event.ThreadID] = methodStacks[event.ThreadID][:i]
			}
		default:
			return speedscope.Output{}, fmt.Errorf(
				"chrometrace: %w: invalid method action: %v",
				errorutil.ErrDataIntegrity,
				event.Action,
			)
		}
	}

	// Close any remaining open frames.
	for threadID, stack := range methodStacks {
		prof := threadIDToProfile[threadID]
		for i := len(stack) - 1; i >= 0; i-- {
			emitEvent(prof, speedscope.EventTypeCloseFrame, stack[i], prof.EndValue)
		}
	}

	allProfiles := make([]interface{}, 0)
	var mainThreadProfileIndex int
	for _, thread := range p.Threads {
		prof, ok := threadIDToProfile[thread.ID]
		if !ok {
			continue
		}
		if thread.Name == mainThread {
			mainThreadProfileIndex = len(allProfiles)
		}
		prof.Name = thread.Name
		allProfiles = append(allProfiles, prof)
	}
	return speedscope.Output{
		ActiveProfileIndex: mainThreadProfileIndex,
		AndroidClock:       string(p.Clock),
		Profiles:           allProfiles,
		Shared:             speedscope.SharedData{Frames: frames},
	}, nil
}

func (p Android) ActiveThreadID() uint64 {
	for _, t := range p.Threads {
		if t.Name == mainThread {
			return t.ID
		}
	}
	return 0
}

func (p Android) GetFrameWithFingerprint(target uint32) (frame.Frame, error) {
	for _, m := range p.Methods {
		f := m.Frame()
		if f.Fingerprint() == target {
			return f, nil
		}
	}
	// TODO: handle react native
	return frame.Frame{}, frame.ErrFrameNotFound
}
