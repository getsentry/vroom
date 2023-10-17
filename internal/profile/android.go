package profile

import (
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
	"github.com/getsentry/vroom/internal/speedscope"
)

type (
	AndroidThread struct {
		ID   uint64 `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	}

	AndroidMethod struct {
		ClassName    string          `json:"class_name,omitempty"`
		Data         Data            `json:"data"`
		ID           uint64          `json:"id,omitempty"`
		InlineFrames []AndroidMethod `json:"inline_frames,omitempty"`
		Name         string          `json:"name,omitempty"`
		Signature    string          `json:"signature,omitempty"`
		SourceFile   string          `json:"source_file,omitempty"`
		SourceLine   uint32          `json:"source_line,omitempty"`
		InApp        *bool           `json:"in_app"`
	}

	Data struct {
		DeobfuscationStatus string `json:"deobfuscation_status,omitempty"`
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
		},
		File:     path.Base(m.SourceFile),
		Function: methodName,
		InApp:    &inApp,
		Line:     m.SourceLine,
		MethodID: m.ID,
		Package:  className,
		Path:     m.SourceFile,
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
		Threads   []AndroidThread `json:"threads,omitempty"`
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

// maxSecs: the highest timestamp/secs in the sequence so far
// latest: the latest time value (at time t-1) before it was updated
// current: current value (at time t) before it's updated
func getAdjustedTime(maxSecs, latest, current uint64) uint64 {
	if current < maxSecs && current < latest {
		return maxSecs + 1
	}
	return maxSecs + (current - latest)
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
	threadMaxSecs := make(map[uint64]uint64)
	threadLatestSampleTime := make(map[uint64]uint64)
	var regression bool

	for i, event := range p.Events {
		current := event.Time.Monotonic.Wall.Secs
		if current < threadLatestSampleTime[event.ThreadID] {
			regression = true
		}

		if regression {
			newSec := getAdjustedTime(threadMaxSecs[event.ThreadID], threadLatestSampleTime[event.ThreadID], current)
			threadMaxSecs[event.ThreadID] = max(threadMaxSecs[event.ThreadID], newSec)

			threadLatestSampleTime[event.ThreadID] = current
			p.Events[i].Time.Monotonic.Wall.Secs = newSec
			continue
		}

		threadLatestSampleTime[event.ThreadID] = current
		threadMaxSecs[event.ThreadID] = max(threadMaxSecs[event.ThreadID], current)
	}
}

// CallTrees generates call trees for a given profile.
func (p Android) CallTrees() map[uint64][]*nodetree.Node {
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

		ts := buildTimestamp(e.Time)
		if ts > maxTimestampNS {
			maxTimestampNS = ts
		}

		switch e.Action {
		case EnterAction:
			enterPerMethod[e.MethodID]++
			m, exists := methods[e.MethodID]
			if !exists {
				methods[e.MethodID] = AndroidMethod{
					ClassName: "unknown",
					ID:        e.MethodID,
					Name:      "unknown",
				}
			}
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

func (p Android) DurationNS() uint64 {
	if len(p.Events) == 0 {
		return 0
	}
	buildTimestamp := p.TimestampGetter()
	startTS := buildTimestamp(p.Events[0].Time)
	endTS := buildTimestamp(p.Events[len(p.Events)-1].Time)
	return endTS - startTS
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

			inApp := inlineMethod.isApplicationFrame(appIdentifier)
			inlineMethod.InApp = &inApp

			method.InlineFrames[j] = inlineMethod
		}

		inApp := method.isApplicationFrame(appIdentifier)
		method.InApp = &inApp

		p.Methods[i] = method
	}
}

func (p Android) Speedscope() (speedscope.Output, error) {
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
	buildTimestamp := p.TimestampGetter()

	enterPerMethod := make(map[uint64]int)
	exitPerMethod := make(map[uint64]int)

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
			enterPerMethod[event.MethodID]++
			methodStacks[event.ThreadID] = append(methodStacks[event.ThreadID], event.MethodID)
			emitEvent(prof, speedscope.EventTypeOpenFrame, event.MethodID, ts)
		case "Exit", "Unwind":
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
				if methodID != event.MethodID &&
					enterPerMethod[event.MethodID] <= exitPerMethod[event.MethodID] {
					eventSkipped = true
					break
				}
				emitEvent(prof, speedscope.EventTypeCloseFrame, methodID, ts)
				exitPerMethod[methodID]++
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
