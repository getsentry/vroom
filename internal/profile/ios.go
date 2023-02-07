package profile

import (
	"fmt"
	"hash"
	"hash/fnv"
	"sort"
	"strconv"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/packageutil"
	"github.com/getsentry/vroom/internal/speedscope"
)

type IosFrame struct {
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

// IsMain returns true if the function is considered the main function.
// It also returns an offset indicate if we need to keep the previous frame or not.
func (f IosFrame) IsMain() (bool, int) {
	if f.Status != "symbolicated" {
		return false, 0
	} else if f.Function == "main" {
		return true, 0
	} else if f.Function == "UIApplicationMain" {
		return true, -1
	}
	return false, 0
}

func (f IosFrame) WriteToHash(h hash.Hash) {
	if f.Package == "" && f.Function == "" {
		h.Write([]byte("-"))
	} else {
		h.Write([]byte(nodetree.PackageBaseName(f.Package)))
		h.Write([]byte(f.Function))
	}
}

func (f IosFrame) Address() string {
	if f.SymAddr != "" {
		return f.SymAddr
	}
	return f.InstructionAddr
}

type Sample struct {
	Frames              []IosFrame `json:"frames,omitempty"`
	Priority            int        `json:"priority,omitempty"`
	QueueAddress        string     `json:"queue_address,omitempty"`
	RelativeTimestampNS uint64     `json:"relative_timestamp_ns,omitempty"`
	State               string     `json:"state,omitempty"`
	ThreadID            uint64     `json:"thread_id,omitempty"`
}

func (s Sample) ContainsMain() bool {
	for i := len(s.Frames) - 1; i >= 0; i-- {
		isMain, _ := s.Frames[i].IsMain()
		if isMain {
			return true
		}
	}
	return false
}

type IOS struct {
	QueueMetadata  map[string]QueueMetadata  `json:"queue_metadata"`
	Samples        []Sample                  `json:"samples"`
	ThreadMetadata map[string]ThreadMetadata `json:"thread_metadata"`
}

type candidate struct {
	ThreadID   uint64
	FrameCount int
}

// MainThread returns what we believe is the main thread ID in the profile
func (p IOS) MainThread() uint64 {
	// Use metadata
	for threadID, m := range p.ThreadMetadata {
		if m.IsMain {
			id, err := strconv.ParseUint(threadID, 10, 64)
			if err != nil {
				break
			}
			return id
		}
	}

	// Check for a main frame
	queues := make(map[uint64]map[QueueMetadata]int)
	for _, s := range p.Samples {
		var isMain bool
		for _, f := range s.Frames {
			if isMain, _ = f.IsMain(); isMain {
				// If we found a frame identified as a main frame, we're sure it's the main thread
				return s.ThreadID
			}
		}
		// Otherwise, we collect queue information to select which queue seems the right one
		if tq, qExists := p.QueueMetadata[s.QueueAddress]; qExists {
			if qm, qmExists := queues[s.ThreadID]; !qmExists {
				queues[s.ThreadID] = make(map[QueueMetadata]int)
			} else {
				frameCount := len(s.Frames)
				if q, qExists := qm[tq]; !qExists {
					qm[tq] = frameCount
				} else if q < frameCount {
					qm[tq] = frameCount
				}
			}
		}
	}
	// Check for the right queue name
	var candidates []candidate
	for threadID, qm := range queues {
		// Only threads with 1 main queue are considered
		if len(qm) == 1 {
			for q, frameCount := range qm {
				if q.LabeledAsMainThread() {
					candidates = append(candidates, candidate{threadID, frameCount})
				}
			}
		}
	}
	// Whoops
	if len(candidates) == 0 {
		return 0
	}
	// Sort possible candidates by deepest stack then lowest thread ID
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].FrameCount == candidates[j].FrameCount {
			return candidates[i].ThreadID < candidates[j].ThreadID
		}
		return candidates[i].FrameCount > candidates[j].FrameCount
	})
	return candidates[0].ThreadID
}

func (p IOS) CallTrees(merge bool) map[uint64][]*nodetree.Node {
	sort.Slice(p.Samples, func(i, j int) bool {
		return p.Samples[i].RelativeTimestampNS < p.Samples[j].RelativeTimestampNS
	})

	activeThreadID := p.MainThread()

	var current *nodetree.Node
	trees := make(map[uint64][]*nodetree.Node)
	h := fnv.New64()
	previousTimestamp := make(map[uint64]uint64)
	for _, s := range p.Samples {
		if s.ThreadID != activeThreadID {
			continue
		}

		frameCount := len(s.Frames)
		// Filter out a bogus root address that appears in some iOS backtraces, this symbol
		// can never be symbolicated and usually contains 1 child.
		if frameCount > 2 && s.Frames[frameCount-1].InstructionAddr == "0xffffffffc" {
			s.Frames = s.Frames[:frameCount-2]
		}
		for i := len(s.Frames) - 1; i >= 0; i-- {
			f := s.Frames[i]
			f.WriteToHash(h)
			fingerprint := h.Sum64()
			if current == nil {
				i := len(trees[s.ThreadID]) - 1
				if i >= 0 && trees[s.ThreadID][i].Fingerprint == fingerprint && trees[s.ThreadID][i].EndNS == previousTimestamp[s.ThreadID] {
					current = trees[s.ThreadID][i]
					current.Update(s.RelativeTimestampNS)
				} else {
					n := nodetree.NodeFromFrame(f.Package, f.Function, f.AbsPath, f.LineNo, previousTimestamp[s.ThreadID], s.RelativeTimestampNS, fingerprint, packageutil.IsIOSApplicationPackage(f.Package))
					trees[s.ThreadID] = append(trees[s.ThreadID], n)
					current = n
				}
			} else {
				i := len(current.Children) - 1
				if i >= 0 && current.Children[i].Fingerprint == fingerprint && current.Children[i].EndNS == previousTimestamp[s.ThreadID] {
					current = current.Children[i]
					current.Update(s.RelativeTimestampNS)
				} else {
					n := nodetree.NodeFromFrame(f.Package, f.Function, f.AbsPath, f.LineNo, previousTimestamp[s.ThreadID], s.RelativeTimestampNS, fingerprint, packageutil.IsIOSApplicationPackage(f.Package))
					current.Children = append(current.Children, n)
					current = n
				}
			}
		}
		h.Reset()
		previousTimestamp[s.ThreadID] = s.RelativeTimestampNS
		current = nil
	}

	return trees
}

func (p IOS) FindNextActiveSample(threadID uint64, i int) Sample {
	for ; i < len(p.Samples); i++ {
		if p.Samples[i].ThreadID == threadID && len(p.Samples[i].Frames) != 0 {
			return p.Samples[i]
		}
	}
	return Sample{}
}

func findCommonFrames(a, b []IosFrame) []IosFrame {
	var c []IosFrame
	for i, j := len(a)-1, len(b)-1; i >= 0 && j >= 0; i, j = i-1, j-1 {
		if a[i].SymAddr == b[j].SymAddr {
			c = append(c, a[i])
			continue
		}
		break
	}
	reverse(c)
	return c
}

func reverse(a []IosFrame) {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}

func (p *IOS) ReplaceIdleStacks() {
	previousActiveSamplePerThreadID := make(map[uint64]int)

	for i, s := range p.Samples {
		if len(s.Frames) != 0 {
			// keep track of the previous active sample as we go
			previousActiveSamplePerThreadID[s.ThreadID] = i
			continue
		}

		// if there's no frame, the thread is considired idle
		p.Samples[i].State = "idle"

		previousSample, exists := previousActiveSamplePerThreadID[s.ThreadID]
		if !exists {
			continue
		}

		previousFrames := p.Samples[previousSample].Frames
		nextFrames := p.FindNextActiveSample(s.ThreadID, i).Frames
		if len(previousFrames) == 0 || len(nextFrames) == 0 {
			continue
		}

		common := findCommonFrames(previousFrames, nextFrames)

		// replace all idle stacks until next active sample
		for j := i; j < len(p.Samples); j++ {
			if p.Samples[j].ThreadID == s.ThreadID && len(p.Samples[j].Frames) == 0 {
				p.Samples[j].Frames = common
				continue
			}
			break
		}
	}
}

type ThreadMetadata struct {
	IsMain   bool   `json:"is_main_thread,omitempty"`
	Name     string `json:"name,omitempty"`
	Priority int    `json:"priority,omitempty"`
}

type QueueMetadata struct {
	Label string `json:"label"`
}

func (q QueueMetadata) LabeledAsMainThread() bool {
	return q.Label == "com.apple.main-thread"
}

func (p IOS) Speedscope() (speedscope.Output, error) {
	threadIDToProfile := make(map[uint64]*speedscope.SampledProfile)
	addressToFrameIndex := make(map[string]int)
	threadIDToPreviousTimestampNS := make(map[uint64]uint64)
	frames := make([]speedscope.Frame, 0)
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
			sampProfile = &speedscope.SampledProfile{
				Name:         threadName,
				Queues:       make(map[string]speedscope.Queue),
				StartValue:   sample.RelativeTimestampNS,
				ThreadID:     sample.ThreadID,
				IsMainThread: sample.ThreadID == mainThreadID,
				Type:         speedscope.ProfileTypeSampled,
				Unit:         speedscope.ValueUnitNanoseconds,
			}
			if qmExists {
				sampProfile.Queues[queueMetadata.Label] = speedscope.Queue{Label: queueMetadata.Label, StartNS: sample.RelativeTimestampNS, EndNS: sample.RelativeTimestampNS}
			}
			if tmExists {
				sampProfile.Priority = threadMetadata.Priority
			}
			threadIDToProfile[sample.ThreadID] = sampProfile
		} else {
			if qmExists {
				q, qExists := sampProfile.Queues[queueMetadata.Label]
				if !qExists {
					sampProfile.Queues[queueMetadata.Label] = speedscope.Queue{Label: queueMetadata.Label, StartNS: sample.RelativeTimestampNS, EndNS: sample.RelativeTimestampNS}
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
				frames = append(frames, speedscope.Frame{
					File:          fr.Filename,
					Image:         nodetree.PackageBaseName(fr.Package),
					IsApplication: packageutil.IsIOSApplicationPackage(fr.Package),
					Line:          fr.LineNo,
					Name:          symbolName,
				})
			}
			samp = append(samp, frameIndex)
		}
		sampProfile.Samples = append(sampProfile.Samples, samp)
	} // end loop speedscope.SampledProfiles
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
