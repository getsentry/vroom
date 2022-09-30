package profile

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/packageutil"
	"github.com/getsentry/vroom/internal/speedscope"
)

type RustFrame struct {
	AbsPath         string `json:"abs_path,omitempty"`
	Filename        string `json:"filename,omitempty"`
	Function        string `json:"function,omitempty"`
	InstructionAddr string `json:"instruction_addr,omitempty"`
	Lang            string `json:"lang,omitempty"`
	LineNo          uint32 `json:"lineno,omitempty"`
	OriginalIndex   int    `json:"original_index,omitempty"`
	Package         string `json:"package"`
	Status          string `json:"status,omitempty"`
	SymAddr         string `json:"sym_addr,omitempty"`
	Symbol          string `json:"symbol,omitempty"`
}

type RustSample struct {
	Frames              []RustFrame `json:"frames,omitempty"`
	RelativeTimestampNS uint64      `json:"nanos_relative_to_start,omitempty"`
	ThreadID            uint64      `json:"thread_id,omitempty"`
	ThreadName          string      `json:"thread_name,omitempty"`
}

type Rust struct {
	StartTimeNS  uint64       `json:"start_time_nanos"`
	StartTimeSec uint64       `json:"start_time_secs"`
	DurationNS   uint64       `json:"duration_nanos"`
	Samples      []RustSample `json:"samples"`
}

// IsMain returns true if the function is considered the main function.
func (f RustFrame) IsMain() bool {
	if f.Status != "symbolicated" {
		return false
	}
	return strings.HasSuffix(f.Function, "::main")
}

// MainThread returns what we believe is the main thread ID in the profile
func (p Rust) MainThread() uint64 {
	// Check for a main frame
	for _, s := range p.Samples {
		var isMain bool
		for _, f := range s.Frames {
			if isMain = f.IsMain(); isMain {
				// If we found a frame identified as a main frame, we're sure it's the main thread
				return s.ThreadID
			}
		}
	}
	return 0
}

func (p Rust) CallTrees() map[uint64][]*nodetree.Node {
	return make(map[uint64][]*nodetree.Node)
}

func (p Rust) Speedscope() (speedscope.Output, error) {
	threadIDToProfile := make(map[uint64]*speedscope.SampledProfile)
	addressToFrameIndex := make(map[string]int)
	threadIDToPreviousTimestampNS := make(map[uint64]uint64)
	frames := make([]speedscope.Frame, 0)
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
			sampProfile = &speedscope.SampledProfile{
				Name:         threadName,
				StartValue:   sample.RelativeTimestampNS,
				ThreadID:     sample.ThreadID,
				IsMainThread: isMainThread,
				Type:         speedscope.ProfileTypeSampled,
				Unit:         speedscope.ValueUnitNanoseconds,
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
				frames = append(frames, speedscope.Frame{
					File:          fr.Filename,
					Image:         nodetree.PackageBaseName(fr.Package),
					Inline:        fr.Status == "symbolicated" && fr.SymAddr == "",
					IsApplication: packageutil.IsRustApplicationPackage(fr.Package),
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

	return speedscope.Output{
		ActiveProfileIndex: mainThreadProfileIndex,
		Profiles:           allProfiles,
		Shared:             speedscope.SharedData{Frames: frames},
	}, nil
}
