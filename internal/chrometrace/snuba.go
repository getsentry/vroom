package chrometrace

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/getsentry/vroom/internal/calltree"
	"github.com/getsentry/vroom/internal/errorutil"
	"github.com/getsentry/vroom/internal/profile"
)

type (
	output struct {
		ActiveProfileIndex int             `json:"activeProfileIndex"`
		AndroidClock       profile.Clock   `json:"androidClock,omitempty"`
		DurationNS         uint64          `json:"durationNS"`
		Metadata           profileMetadata `json:"metadata"`
		Platform           string          `json:"platform"`
		ProfileID          string          `json:"profileID"`
		Profiles           []interface{}   `json:"profiles"`
		ProjectID          uint64          `json:"projectID"`
		Shared             sharedData      `json:"shared"`
		TransactionName    string          `json:"transactionName"`
		Version            string          `json:"version"`
	}

	profileMetadata struct {
		profileView

		Version string `json:"version"`
	}

	profileView struct {
		AndroidAPILevel      uint32        `json:"androidAPILevel,omitempty"`
		Architecture         string        `json:"architecture,omitempty"`
		DebugMeta            interface{}   `json:"-"`
		DeviceClassification string        `json:"deviceClassification"`
		DeviceLocale         string        `json:"deviceLocale"`
		DeviceManufacturer   string        `json:"deviceManufacturer"`
		DeviceModel          string        `json:"deviceModel"`
		DeviceOSBuildNumber  string        `json:"deviceOSBuildNumber,omitempty"`
		DeviceOSName         string        `json:"deviceOSName"`
		DeviceOSVersion      string        `json:"deviceOSVersion"`
		DurationNS           uint64        `json:"durationNS"`
		Environment          string        `json:"environment,omitempty"`
		OrganizationID       uint64        `json:"organizationID"`
		Platform             string        `json:"platform"`
		Profile              string        `json:"-"`
		ProfileID            string        `json:"profileID"`
		ProjectID            uint64        `json:"projectID"`
		Received             time.Time     `json:"received"`
		Trace                profile.Trace `json:"-"`
		TraceID              string        `json:"traceID"`
		TransactionID        string        `json:"transactionID"`
		TransactionName      string        `json:"transactionName"`
		VersionCode          string        `json:"-"`
		VersionName          string        `json:"-"`
	}
)

func (o *output) SetVersion() {
	version := profile.FormatVersion(o.Metadata.VersionName, o.Metadata.VersionCode)
	o.Version, o.Metadata.Version = version, version
}

// SpeedscopeFromSnuba generates a profile using the Speedscope format from data in Snuba
func SpeedscopeFromSnuba(p profile.LegacyProfile) ([]byte, error) {
	var o output
	switch p.Platform {
	case "android":
		var androidProfile profile.Android
		err := json.Unmarshal([]byte(p.Profile), &androidProfile)
		if err != nil {
			return nil, err
		}
		o, err = androidSpeedscopeTraceFromProfile(&androidProfile)
		if err != nil {
			return nil, err
		}
	case "cocoa":
		var iosProfile profile.IOS
		err := json.Unmarshal([]byte(p.Profile), &iosProfile)
		if err != nil {
			return nil, err
		}
		iosProfile.ReplaceIdleStacks()
		o = iosSpeedscopeTraceFromProfile(&iosProfile)
		if err != nil {
			return nil, err
		}
	case "rust":
		var rustProfile profile.Rust
		err := json.Unmarshal([]byte(p.Profile), &rustProfile)
		if err != nil {
			return nil, err
		}
		o = rustSpeedscopeTraceFromProfile(&rustProfile)
		if err != nil {
			return nil, err
		}
	case "python":
		var pythonProfile profile.Python
		err := json.Unmarshal([]byte(p.Profile), &pythonProfile)
		if err != nil {
			return nil, err
		}
		o = pythonSpeedscopeTraceFromProfile(&pythonProfile)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("chrometrace: %w: %s is not a supported platform", errorutil.ErrDataIntegrity, p.Platform)
	}
	o.DurationNS = p.DurationNS
	o.Metadata = profileMetadata{profileView: profileView(p)}
	o.Platform = p.Platform
	o.ProfileID = p.ProfileID
	o.ProjectID = p.ProjectID
	o.TransactionName = p.TransactionName

	o.SetVersion()

	return json.Marshal(o)
}

func iosSpeedscopeTraceFromProfile(p *profile.IOS) output {
	threadIDToProfile := make(map[uint64]*sampledProfile)
	addressToFrameIndex := make(map[string]int)
	threadIDToPreviousTimestampNS := make(map[uint64]uint64)
	frames := make([]frame, 0)
	// we need to find the frame index of the main function so we can remove the frames before it
	mainFunctionFrameIndex := -1
	mainThreadID := p.MainThread()
	for _, sample := range p.Samples {
		threadID := strconv.FormatUint(sample.ThreadID, 10)
		sampProfile, ok := threadIDToProfile[sample.ThreadID]
		queueMetadata, qmExists := p.QueueMetadata[sample.QueueAddress]
		if !ok {
			threadMetadata, tmExists := p.ThreadMetadata[threadID]
			threadName := threadMetadata.Name
			if threadName == "" && qmExists && (!queueMetadata.LabeledAsMainThread() || sample.ThreadID != mainThreadID) {
				threadName = queueMetadata.Label
			}
			sampProfile = &sampledProfile{
				Name:         threadName,
				Queues:       make(map[string]queue),
				StartValue:   sample.RelativeTimestampNS,
				ThreadID:     sample.ThreadID,
				IsMainThread: sample.ThreadID == mainThreadID,
				Type:         profileTypeSampled,
				Unit:         valueUnitNanoseconds,
			}
			if qmExists {
				sampProfile.Queues[queueMetadata.Label] = queue{Label: queueMetadata.Label, StartNS: sample.RelativeTimestampNS, EndNS: sample.RelativeTimestampNS}
			}
			if tmExists {
				sampProfile.Priority = threadMetadata.Priority
			}
			threadIDToProfile[sample.ThreadID] = sampProfile
		} else {
			if qmExists {
				q, qExists := sampProfile.Queues[queueMetadata.Label]
				if !qExists {
					sampProfile.Queues[queueMetadata.Label] = queue{Label: queueMetadata.Label, StartNS: sample.RelativeTimestampNS, EndNS: sample.RelativeTimestampNS}
				} else {
					q.EndNS = sample.RelativeTimestampNS
					sampProfile.Queues[queueMetadata.Label] = q
				}
			}
			sampProfile.Weights = append(sampProfile.Weights, sample.RelativeTimestampNS-threadIDToPreviousTimestampNS[sample.ThreadID])
		}

		sampProfile.EndValue = sample.RelativeTimestampNS
		threadIDToPreviousTimestampNS[sample.ThreadID] = sample.RelativeTimestampNS

		samp := make([]int, 0, len(sample.Frames))
		for i := len(sample.Frames) - 1; i >= 0; i-- {
			fr := sample.Frames[i]
			address := fr.Address()
			frameIndex, ok := addressToFrameIndex[address]
			if !ok {
				frameIndex = len(frames)
				symbolName := fr.Function
				if symbolName == "" {
					symbolName = fmt.Sprintf("unknown (%s)", address)
				} else if mainFunctionFrameIndex == -1 {
					if isMainFrame, i := fr.IsMain(); isMainFrame {
						mainFunctionFrameIndex = frameIndex + i
					}
				}
				addressToFrameIndex[address] = frameIndex
				frames = append(frames, frame{
					File:          fr.Filename,
					Image:         calltree.ImageBaseName(fr.Package),
					IsApplication: profile.IsIOSApplicationImage(fr.Package),
					Line:          fr.LineNo,
					Name:          symbolName,
				})
			}
			samp = append(samp, frameIndex)
		}
		sampProfile.Samples = append(sampProfile.Samples, samp)
	} // end loop sampledProfiles
	var mainThreadProfileIndex int
	allProfiles := make([]interface{}, 0)
	for _, prof := range threadIDToProfile {
		if prof.IsMainThread {
			mainThreadProfileIndex = len(allProfiles)
			// Remove all frames before main is called on the main thread
			if mainFunctionFrameIndex != -1 {
				for i, sample := range prof.Samples {
					for j, frameIndex := range sample {
						if frameIndex == mainFunctionFrameIndex {
							prof.Samples[i] = prof.Samples[i][j:]
							break
						}
					}
				}
			}
		}
		prof.Weights = append(prof.Weights, 0)
		allProfiles = append(allProfiles, prof)
	}
	return output{
		ActiveProfileIndex: mainThreadProfileIndex,
		Profiles:           allProfiles,
		Shared:             sharedData{Frames: frames},
	}
}

func androidSpeedscopeTraceFromProfile(p *profile.Android) (output, error) {
	frames := make([]frame, 0)
	methodIDToFrameIndex := make(map[uint64][]int)
	for _, method := range p.Methods {
		if len(method.InlineFrames) > 0 {
			for _, m := range method.InlineFrames {
				methodIDToFrameIndex[method.ID] = append(methodIDToFrameIndex[method.ID], len(frames))
				frames = append(frames, frame{
					File:          m.SourceFile,
					Image:         m.ClassName,
					Inline:        true,
					IsApplication: !profile.IsAndroidSystemPackage(m.ClassName),
					Line:          m.SourceLine,
					Name:          m.Name,
				})

			}
		} else {
			packageName, _, err := method.ExtractPackageNameAndSimpleMethodNameFromAndroidMethod()
			if err != nil {
				return output{}, err
			}
			fullMethodName, err := method.FullMethodNameFromAndroidMethod()
			if err != nil {
				return output{}, err
			}
			methodIDToFrameIndex[method.ID] = append(methodIDToFrameIndex[method.ID], len(frames))
			frames = append(frames, frame{
				Name:          fullMethodName,
				File:          method.SourceFile,
				Line:          method.SourceLine,
				IsApplication: !profile.IsAndroidSystemPackage(fullMethodName),
				Image:         packageName,
			})
		}
	}

	emitEvent := func(p *eventedProfile, et eventType, methodID, ts uint64) {
		frameIndexes, ok := methodIDToFrameIndex[methodID]
		if !ok {
			// sometimes it might happen that a method is listed in events but an entry definition
			// is not correctly defined in the methods entry. We don't wan't to fail the whole chrometrace
			// for this so we create a method on the fly
			frameIndexes = []int{len(frames)}
			methodIDToFrameIndex[methodID] = append(methodIDToFrameIndex[methodID], frameIndexes[0])
			frames = append(frames, frame{
				Name:          fmt.Sprintf("unknown (id %d)", methodID),
				File:          "unknown",
				Line:          0,
				IsApplication: false,
				Image:         "unknown",
			})
		}
		for _, fi := range frameIndexes {
			p.Events = append(p.Events, event{
				Type:  et,
				Frame: fi,
				At:    ts,
			})
		}
	}

	threadIDToProfile := make(map[uint64]*eventedProfile)
	methodStacks := make(map[uint64][]uint64) // map of thread ID -> stack of method IDs
	buildTimestamp := p.TimestampGetter()

	for _, event := range p.Events {
		ts := buildTimestamp(event.Time)
		prof, ok := threadIDToProfile[event.ThreadID]
		if !ok {
			threadID := event.ThreadID
			prof = &eventedProfile{
				StartValue: ts,
				ThreadID:   threadID,
				Type:       profileTypeEvented,
				Unit:       valueUnitNanoseconds,
			}
			threadIDToProfile[threadID] = prof
		}
		prof.EndValue = ts

		switch event.Action {
		case "Enter":
			methodStacks[event.ThreadID] = append(methodStacks[event.ThreadID], event.MethodID)
			emitEvent(prof, eventTypeOpenFrame, event.MethodID, ts)
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
			for ; i >= 0; i-- {
				methodID := stack[i]
				emitEvent(prof, eventTypeCloseFrame, methodID, ts)

				if methodID == event.MethodID {
					break
				}
			}
			if stack[i] != event.MethodID {
				return output{}, fmt.Errorf("chrometrace: %w: ending event %v but stack for thread %v does not contain that record", errorutil.ErrDataIntegrity, event, event.ThreadID)
			}
			// Pop the elements that we emitted end events for off the stack
			methodStacks[event.ThreadID] = methodStacks[event.ThreadID][:i]

		default:
			return output{}, fmt.Errorf("chrometrace: %w: invalid method action: %v", errorutil.ErrDataIntegrity, event.Action)
		} // end switch
	} // end loop events

	// Close any remaining open frames.
	for threadID, stack := range methodStacks {
		prof := threadIDToProfile[threadID]
		for i := len(stack) - 1; i >= 0; i-- {
			emitEvent(prof, eventTypeCloseFrame, stack[i], prof.EndValue)
		}
	}

	allProfiles := make([]interface{}, 0)
	var mainThreadProfileIndex int
	for _, thread := range p.Threads {
		prof, ok := threadIDToProfile[thread.ID]
		if !ok {
			continue
		}
		if thread.Name == "main" {
			mainThreadProfileIndex = len(allProfiles)
		}
		prof.Name = thread.Name
		allProfiles = append(allProfiles, prof)
	}
	return output{
		ActiveProfileIndex: mainThreadProfileIndex,
		AndroidClock:       p.Clock,
		Profiles:           allProfiles,
		Shared:             sharedData{Frames: frames},
	}, nil
}

func pythonSpeedscopeTraceFromProfile(p *profile.Python) output {
	threadIDToProfile := make(map[uint64]*sampledProfile)
	threadIDToPreviousTimestampNS := make(map[uint64]uint64)

	sort.Slice(p.Samples, func(i, j int) bool {
		return p.Samples[i].RelativeTimestampNS <= p.Samples[j].RelativeTimestampNS
	})
	for _, sample := range p.Samples {
		sampProfile, ok := threadIDToProfile[sample.ThreadID]
		if !ok {
			sampProfile = &sampledProfile{
				StartValue:   sample.RelativeTimestampNS,
				ThreadID:     sample.ThreadID,
				IsMainThread: false,
				Type:         profileTypeSampled,
				Unit:         valueUnitNanoseconds,
			}
			threadIDToProfile[sample.ThreadID] = sampProfile
		} else {
			sampProfile.Weights = append(sampProfile.Weights, sample.RelativeTimestampNS-threadIDToPreviousTimestampNS[sample.ThreadID])
		}

		samp := make([]int, 0, len(sample.Frames))
		for i := len(sample.Frames) - 1; i >= 0; i-- {
			samp = append(samp, sample.Frames[i])
		}

		sampProfile.Samples = append(sampProfile.Samples, samp)
		sampProfile.EndValue = sample.RelativeTimestampNS
		threadIDToPreviousTimestampNS[sample.ThreadID] = sample.RelativeTimestampNS
	}

	frames := make([]frame, 0, len(p.Frames))
	for _, pythonFrame := range p.Frames {
		frames = append(frames, frame{
			File: pythonFrame.File,
			Name: pythonFrame.Name,
			Line: pythonFrame.Line,
		})
	}

	var mainThreadProfileIndex int
	var mainThreadID uint64

	allProfiles := make([]interface{}, 0)
	for threadID, prof := range threadIDToProfile {
		// There is no thread metadata being sent by the python profiler at the moment,
		// so we use this heuristic to find a main thread. not perfect but good enough
		// until we start sending the metadata needed.
		if threadID < mainThreadID {
			mainThreadID = threadID
		}
		prof.Weights = append(prof.Weights, 0)
		allProfiles = append(allProfiles, prof)
	}

	return output{
		ActiveProfileIndex: mainThreadProfileIndex,
		Profiles:           allProfiles,
		Shared:             sharedData{Frames: frames},
	}
}

func rustSpeedscopeTraceFromProfile(p *profile.Rust) output {
	threadIDToProfile := make(map[uint64]*sampledProfile)
	addressToFrameIndex := make(map[string]int)
	threadIDToPreviousTimestampNS := make(map[uint64]uint64)
	frames := make([]frame, 0)
	// we need to find the frame index of the main function so we can remove the frames before it
	mainFunctionFrameIndex := -1
	mainThreadID := p.MainThread()
	// sorting here is necessary because the timing info for each sample is given by
	// a Rust SystemTime type, which is measurement of the system clock and is not monotonic
	//
	// see: https://doc.rust-lang.org/std/time/struct.SystemTime.html
	sort.Slice(p.Samples, func(i, j int) bool {
		return p.Samples[i].RelativeTimestampNS <= p.Samples[j].RelativeTimestampNS
	})
	for _, sample := range p.Samples {
		sampProfile, ok := threadIDToProfile[sample.ThreadID]
		if !ok {
			isMainThread := sample.ThreadID == mainThreadID

			// the rust profiler automatically use thread_id as a thread_name
			// when the thread_name is not available.
			// So if thread_name == mainThreadID we now it's the main thread
			// and we can replace it with `main`
			var threadName string
			if threadName != strconv.FormatUint(sample.ThreadID, 10) {
				threadName = sample.ThreadName
			} else if isMainThread {
				threadName = "main"
			}
			sampProfile = &sampledProfile{
				Name:         threadName,
				StartValue:   sample.RelativeTimestampNS,
				ThreadID:     sample.ThreadID,
				IsMainThread: isMainThread,
				Type:         profileTypeSampled,
				Unit:         valueUnitNanoseconds,
			}
			threadIDToProfile[sample.ThreadID] = sampProfile
		} else {
			sampProfile.Weights = append(sampProfile.Weights, sample.RelativeTimestampNS-threadIDToPreviousTimestampNS[sample.ThreadID])
		}

		sampProfile.EndValue = sample.RelativeTimestampNS
		threadIDToPreviousTimestampNS[sample.ThreadID] = sample.RelativeTimestampNS
		samp := make([]int, 0, len(sample.Frames))
		for i := len(sample.Frames) - 1; i >= 0; i-- {
			fr := sample.Frames[i]
			var address string
			if fr.SymAddr != "" {
				address = fr.SymAddr
			} else {
				address = fr.InstructionAddr
			}
			frameIndex, ok := addressToFrameIndex[address]
			if !ok {
				frameIndex = len(frames)
				symbolName := fr.Function
				if symbolName == "" {
					symbolName = fmt.Sprintf("unknown (%s)", fr.InstructionAddr)
				} else if mainFunctionFrameIndex == -1 {
					if isMainFrame := fr.IsMain(); isMainFrame {
						mainFunctionFrameIndex = frameIndex
					}
				}
				addressToFrameIndex[address] = frameIndex
				frames = append(frames, frame{
					File:          fr.Filename,
					Image:         calltree.ImageBaseName(fr.Package),
					Inline:        fr.Status == "symbolicated" && fr.SymAddr == "",
					IsApplication: profile.IsRustApplicationImage(fr.Package),
					Line:          fr.LineNo,
					Name:          symbolName,
				})
			}
			samp = append(samp, frameIndex)
		}
		sampProfile.Samples = append(sampProfile.Samples, samp)
	} // end loop sampledProfiles

	var mainThreadProfileIndex int
	allProfiles := make([]interface{}, 0)
	for _, prof := range threadIDToProfile {
		if prof.IsMainThread {
			mainThreadProfileIndex = len(allProfiles)
			// Remove all frames before main is called on the main thread
			if mainFunctionFrameIndex != -1 {
				for i, sample := range prof.Samples {
					for j, frameIndex := range sample {
						if frameIndex == mainFunctionFrameIndex {
							prof.Samples[i] = prof.Samples[i][j:]
							break
						}
					}
				}
			}
		}
		prof.Weights = append(prof.Weights, 0)
		allProfiles = append(allProfiles, prof)
	}

	return output{
		ActiveProfileIndex: mainThreadProfileIndex,
		Profiles:           allProfiles,
		Shared:             sharedData{Frames: frames},
	}
}
