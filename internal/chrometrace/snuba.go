package chrometrace

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/getsentry/vroom/internal/aggregate"
	"github.com/getsentry/vroom/internal/android"
	"github.com/getsentry/vroom/internal/calltree"
	"github.com/getsentry/vroom/internal/errorutil"
	"github.com/getsentry/vroom/internal/snubautil"
)

type output struct {
	ActiveProfileIndex int           `json:"activeProfileIndex"`
	AndroidClock       android.Clock `json:"androidClock,omitempty"`
	DurationNS         uint64        `json:"durationNS"`
	Platform           string        `json:"platform"`
	ProfileID          string        `json:"profileID"`
	Profiles           []interface{} `json:"profiles"`
	ProjectID          uint64        `json:"projectID"`
	Shared             sharedData    `json:"shared"`
	TransactionName    string        `json:"transactionName"`
	Version            string        `json:"version"`
}

// SpeedscopeFromSnuba generates a profile using the Speedscope format from data in Snuba
func SpeedscopeFromSnuba(profile snubautil.Profile) ([]byte, error) {
	var p output
	switch profile.Platform {
	case "android":
		var androidProfile android.AndroidProfile
		err := json.Unmarshal([]byte(profile.Profile), &androidProfile)
		if err != nil {
			return nil, err
		}
		p, err = androidSpeedscopeTraceFromProfile(&androidProfile)
		if err != nil {
			return nil, err
		}
	case "cocoa":
		var iosProfile aggregate.IosProfile
		err := json.Unmarshal([]byte(profile.Profile), &iosProfile)
		if err != nil {
			return nil, err
		}
		p, err = iosSpeedscopeTraceFromProfile(&iosProfile)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("chrometrace: %w: %s is not a supported platform", errorutil.ErrDataIntegrity, profile.Platform)
	}
	p.DurationNS = profile.DurationNs
	p.Platform = profile.Platform
	p.ProfileID = profile.ProfileID
	p.ProjectID = profile.ProjectID
	p.TransactionName = profile.TransactionName
	p.Version = profile.Version()
	return json.Marshal(p)
}

func iosSpeedscopeTraceFromProfile(profile *aggregate.IosProfile) (output, error) {
	threadIDToProfile := make(map[uint64]*sampledProfile)
	addressToFrameIndex := make(map[string]int)
	threadIDToPreviousTimestampNS := make(map[uint64]uint64)
	frames := make([]frame, 0)
	// we need to find the frame index of the main function so we can remove the frames before it
	mainFunctionFrameIndex := -1
	for _, sample := range profile.Samples {
		var threadID uint64
		switch v := sample.ThreadID.(type) {
		case string:
			var err error
			threadID, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				return output{}, err
			}
		case float64:
			threadID = uint64(v)
		case uint64:
			threadID = v
		default:
			return output{}, fmt.Errorf("unknown threadID value type: %T for %v", v, v)
		}
		var relativeTimestampNS uint64
		switch v := sample.RelativeTimestampNS.(type) {
		case string:
			var err error
			relativeTimestampNS, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				return output{}, err
			}
		case float64:
			relativeTimestampNS = uint64(v)
		case uint64:
			relativeTimestampNS = v
		default:
			return output{}, fmt.Errorf("unknown relativeTimestampNS value type: %T for %v", v, v)
		}
		sampProfile, ok := threadIDToProfile[threadID]
		if !ok {
			threadMetadata, tmExists := profile.ThreadMetadata[strconv.FormatUint(threadID, 10)]
			queueMetadata, qmExists := profile.QueueMetadata[sample.QueueAddress]
			threadName := threadMetadata.Name
			if threadName == "" {
				threadName = queueMetadata.Label
			}
			sampProfile = &sampledProfile{
				IsMainThread: threadName == iosMainThreadName,
				Name:         threadName,
				Queues:       make(map[string]queue),
				StartValue:   relativeTimestampNS,
				ThreadID:     threadID,
				Type:         profileTypeSampled,
				Unit:         valueUnitNanoseconds,
			}
			if qmExists {
				sampProfile.Queues[queueMetadata.Label] = queue{Label: queueMetadata.Label, StartNS: relativeTimestampNS, EndNS: relativeTimestampNS}
			}
			if tmExists {
				sampProfile.Priority = threadMetadata.Priority
			}
			threadIDToProfile[threadID] = sampProfile
		} else {
			queueMetadata, qmExists := profile.QueueMetadata[sample.QueueAddress]
			if qmExists {
				q, qExists := sampProfile.Queues[queueMetadata.Label]
				if !qExists {
					sampProfile.Queues[queueMetadata.Label] = queue{Label: queueMetadata.Label, StartNS: relativeTimestampNS, EndNS: relativeTimestampNS}
				} else {
					q.EndNS = relativeTimestampNS
					sampProfile.Queues[queueMetadata.Label] = q
				}
			}
			sampProfile.Weights = append(sampProfile.Weights, relativeTimestampNS-threadIDToPreviousTimestampNS[threadID])
		}

		sampProfile.EndValue = relativeTimestampNS
		threadIDToPreviousTimestampNS[threadID] = relativeTimestampNS

		samp := make([]int, 0, len(sample.Frames))
		for i := len(sample.Frames) - 1; i >= 0; i-- {
			fr := sample.Frames[i]
			// the main thread may not always have the correct name if the thread
			// contains the main function, we should consider it the main thread too
			if !sampProfile.IsMainThread && fr.Function == iosMainFunctionName {
				sampProfile.IsMainThread = true
			}
			frameIndex, ok := addressToFrameIndex[fr.InstructionAddr]
			if !ok {
				frameIndex = len(frames)
				symbolName := fr.Function
				if symbolName == "" {
					symbolName = fmt.Sprintf("unknown (%s)", fr.InstructionAddr)
				} else if symbolName == "main" {
					mainFunctionFrameIndex = frameIndex
				}
				addressToFrameIndex[fr.InstructionAddr] = frameIndex
				frames = append(frames, frame{
					File:          fr.Filename,
					Image:         calltree.ImageBaseName(fr.Package),
					IsApplication: aggregate.IsIOSApplicationImage(fr.Package),
					Line:          fr.LineNo,
					Name:          symbolName,
				})
			}
			samp = append(samp, frameIndex)
		}
		sampProfile.Samples = append(sampProfile.Samples, samp)
	} // end loop sampledProfiles

	threadIDs := make([]uint64, 0, len(threadIDToProfile))
	for threadID := range threadIDToProfile {
		threadIDs = append(threadIDs, threadID)
	}
	sort.SliceStable(threadIDs, func(i, j int) bool {
		return threadIDs[i] < threadIDs[j]
	})

	var mainThreadProfileIndex int
	mainThreadSamples := 0
	allProfiles := make([]interface{}, 0)
	for _, threadID := range threadIDs {
		prof, ok := threadIDToProfile[threadID]
		if !ok {
			continue
		}
		if prof.IsMainThread {

			// There are multiple threads that can be marked as
			// the main thread at times. We should favor the one
			// with the most amount of samples to be shown first.
			if len(prof.Samples) > mainThreadSamples {
				mainThreadProfileIndex = len(allProfiles)
				mainThreadSamples = len(prof.Samples)
			}

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
	}, nil
}

func androidSpeedscopeTraceFromProfile(profile *android.AndroidProfile) (output, error) {
	frames := make([]frame, 0)
	methodIDToFrameIndex := make(map[uint64][]int)
	for _, method := range profile.Methods {
		if len(method.InlineFrames) > 0 {
			for _, m := range method.InlineFrames {
				methodIDToFrameIndex[method.ID] = append(methodIDToFrameIndex[method.ID], len(frames))
				frames = append(frames, frame{
					Name:          m.Name,
					File:          m.SourceFile,
					Line:          int(m.SourceLine),
					IsApplication: !aggregate.IsAndroidSystemPackage(m.ClassName),
					Image:         m.ClassName,
				})

			}
		} else {
			packageName, _, err := android.ExtractPackageNameAndSimpleMethodNameFromAndroidMethod(&method)
			if err != nil {
				return output{}, err
			}
			fullMethodName, err := android.FullMethodNameFromAndroidMethod(&method)
			if err != nil {
				return output{}, err
			}
			methodIDToFrameIndex[method.ID] = append(methodIDToFrameIndex[method.ID], len(frames))
			frames = append(frames, frame{
				Name:          fullMethodName,
				File:          method.SourceFile,
				Line:          int(method.SourceLine),
				IsApplication: !aggregate.IsAndroidSystemPackage(fullMethodName),
				Image:         packageName,
			})
		}
	}

	emitEvent := func(profile *eventedProfile, et eventType, methodID, ts uint64) error {
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
			profile.Events = append(profile.Events, event{
				Type:  et,
				Frame: fi,
				At:    ts,
			})
		}
		return nil
	}

	threadIDToProfile := make(map[uint64]*eventedProfile)
	methodStacks := make(map[uint64][]uint64) // map of thread ID -> stack of method IDs

	var buildTimestamp func(t android.EventTime) uint64

	switch profile.Clock {
	case android.GlobalClock:
		buildTimestamp = func(t android.EventTime) uint64 {
			return t.Global.Secs*uint64(time.Second) + t.Global.Nanos - profile.StartTime
		}
	case android.CPUClock:
		buildTimestamp = func(t android.EventTime) uint64 {
			return t.Monotonic.Cpu.Secs*uint64(time.Second) + t.Monotonic.Cpu.Nanos
		}
	default:
		buildTimestamp = func(t android.EventTime) uint64 {
			return t.Monotonic.Wall.Secs*uint64(time.Second) + t.Monotonic.Wall.Nanos
		}
	}

	for _, event := range profile.Events {
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
			if err := emitEvent(prof, eventTypeOpenFrame, event.MethodID, ts); err != nil {
				return output{}, err
			}
		case "Exit", "Unwind":
			stack := methodStacks[event.ThreadID]
			if len(stack) == 0 {
				return output{}, fmt.Errorf("chrometrace: %w: ending event %v but stack for thread %v is empty", errorutil.ErrDataIntegrity, event, event.ThreadID)
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
				if err := emitEvent(prof, eventTypeCloseFrame, methodID, ts); err != nil {
					return output{}, err
				}
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
			if err := emitEvent(prof, eventTypeCloseFrame, stack[i], prof.EndValue); err != nil {
				return output{}, err
			}
		}
	}

	allProfiles := make([]interface{}, 0)
	var mainThreadProfileIndex int
	for _, thread := range profile.Threads {
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
		AndroidClock:       profile.Clock,
		Profiles:           allProfiles,
		Shared:             sharedData{Frames: frames},
	}, nil
}
