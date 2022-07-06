package aggregate

import (
	"encoding/binary"
	"hash/fnv"
	"sort"

	"github.com/getsentry/vroom/internal/nodetree"
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

type Sample struct {
	Frames              []IosFrame `json:"frames,omitempty"`
	Priority            int        `json:"priority,omitempty"`
	QueueAddress        string     `json:"queue_address,omitempty"`
	RelativeTimestampNS uint64     `json:"relative_timestamp_ns,omitempty"`
	ThreadID            uint64     `json:"thread_id,omitempty"`
}

func (s Sample) ContainsMain() bool {
	i := sort.Search(len(s.Frames), func(i int) bool {
		f := s.Frames[i]
		isMain, _ := f.IsMain()
		return isMain
	})
	return i < len(s.Frames)
}

type IosProfile struct {
	QueueMetadata  map[string]QueueMetadata `json:"queue_metadata"`
	Samples        []Sample                 `json:"samples"`
	ThreadMetadata map[string]ThreadMedata  `json:"thread_metadata"`
}

type candidate struct {
	ThreadID   uint64
	FrameCount int
}

// MainThread returns what we believe is the main thread ID in the profile
func (p IosProfile) MainThread() uint64 {
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
				if q.IsMainThread() {
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

func (p IosProfile) CallTrees() map[uint64][]*nodetree.Node {
	sort.Slice(p.Samples, func(i, j int) bool {
		return p.Samples[i].RelativeTimestampNS < p.Samples[j].RelativeTimestampNS
	})

	var current *nodetree.Node
	trees := make(map[uint64][]*nodetree.Node)
	fingerprint := fnv.New64()
	previousTimestamp := make(map[uint64]uint64)
	buffer := make([]byte, 8)
	for _, s := range p.Samples {
		binary.LittleEndian.PutUint64(buffer, s.ThreadID)
		fingerprint.Write(buffer)
		for i := len(s.Frames) - 1; i >= 0; i-- {
			f := s.Frames[i]
			fingerprint.Write([]byte(f.InstructionAddr))
			id := fingerprint.Sum64()
			if current == nil {
				for _, r := range trees[s.ThreadID] {
					if r.ID == id {
						r.SetDuration(s.RelativeTimestampNS)
						current = r
						break
					}
				}
				if current == nil {
					n := nodetree.NodeFromFrame(f.Package, f.Symbol, f.AbsPath, f.LineNo, previousTimestamp[s.ThreadID], s.RelativeTimestampNS, id)
					trees[s.ThreadID] = append(trees[s.ThreadID], n)
					current = n
				}
			} else {
				count := len(current.Children)
				if count > 0 && current.Children[count-1].ID == id {
					current.Children[count-1].SetDuration(s.RelativeTimestampNS)
					current = current.Children[count-1]
				} else {
					parentTimestamp := previousTimestamp[s.ThreadID]
					if count > 0 {
						parentTimestamp = current.Children[count-1].EndNS
					}
					n := nodetree.NodeFromFrame(f.Package, f.Symbol, f.AbsPath, f.LineNo, parentTimestamp, s.RelativeTimestampNS, id)
					current.Children = append(current.Children, n)
					current = n
				}
			}
		}
		fingerprint.Reset()
		previousTimestamp[s.ThreadID] = s.RelativeTimestampNS
		current = nil
	}

	return trees
}

type ThreadMedata struct {
	Name     string `json:"name"`
	Priority int    `json:"priority"`
}

type QueueMetadata struct {
	Label string `json:"label"`
}

func (q QueueMetadata) IsMainThread() bool {
	return q.Label == "com.apple.main-thread"
}