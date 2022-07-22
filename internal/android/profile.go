package android

import (
	"hash/fnv"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/nodetree"
)

type AndroidThread struct {
	ID   uint64 `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type AndroidMethod struct {
	ClassName    string          `json:"class_name,omitempty"`
	ID           uint64          `json:"id,omitempty"`
	InlineFrames []AndroidMethod `json:"inline_frames,omitempty"`
	Name         string          `json:"name,omitempty"`
	Signature    string          `json:"signature,omitempty"`
	SourceFile   string          `json:"source_file,omitempty"`
	SourceLine   uint32          `json:"source_line,omitempty"`
}

type EventMonotonic struct {
	Wall Duration `json:"wall,omitempty"`
	Cpu  Duration `json:"cpu,omitempty"`
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

type AndroidProfile struct {
	Clock     Clock           `json:"clock"`
	Events    []AndroidEvent  `json:"events,omitempty"`
	Methods   []AndroidMethod `json:"methods,omitempty"`
	StartTime uint64          `json:"start_time,omitempty"`
	Threads   []AndroidThread `json:"threads,omitempty"`
}

type Clock string

const (
	DualClock   Clock = "Dual"
	CPUClock    Clock = "Cpu"
	WallClock   Clock = "Wall"
	GlobalClock Clock = "Global"
)

func (p AndroidProfile) TimestampGetter() func(EventTime) uint64 {
	var buildTimestamp func(t EventTime) uint64
	switch p.Clock {
	case GlobalClock:
		buildTimestamp = func(t EventTime) uint64 {
			return t.Global.Secs*uint64(time.Second) + t.Global.Nanos - p.StartTime
		}
	case CPUClock:
		buildTimestamp = func(t EventTime) uint64 {
			return t.Monotonic.Cpu.Secs*uint64(time.Second) + t.Monotonic.Cpu.Nanos
		}
	default:
		buildTimestamp = func(t EventTime) uint64 {
			return t.Monotonic.Wall.Secs*uint64(time.Second) + t.Monotonic.Wall.Nanos
		}
	}
	return buildTimestamp
}

func (p AndroidProfile) CallTrees() map[uint64][]*nodetree.Node {
	buildTimestamp := p.TimestampGetter()
	trees := make(map[uint64][]*nodetree.Node)
	stacks := make(map[uint64][]*nodetree.Node)
	methods := make(map[uint64]AndroidMethod)
	for _, m := range p.Methods {
		methods[m.ID] = m
	}
	for _, e := range p.Events {
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
			n := nodetree.NodeFromFrame(m.ClassName, m.Name, m.SourceFile, m.SourceLine, buildTimestamp(e.Time), 0, 0, !IsSystemPackage(m.ClassName))
			if len(stacks[e.ThreadID]) == 0 {
				trees[e.ThreadID] = append(trees[e.ThreadID], n)
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
			n := stacks[e.ThreadID][i]
			n.DurationNS = buildTimestamp(e.Time) - n.StartNS
			stacks[e.ThreadID] = stacks[e.ThreadID][:i]
		}
	}

	return trees
}

func generateFingerprint(stack []*nodetree.Node) uint64 {
	h := fnv.New64()
	for _, n := range stack {
		n.WriteToHash(h)
	}
	return h.Sum64()
}

var (
	androidPackagePrefixes = []string{
		"android.",
		"androidx.",
		"com.android.",
		"com.google.android.",
		"com.motorola.",
		"java.",
		"javax.",
		"kotlin.",
		"kotlinx.",
		"retrofit2.",
		"sun.",
	}
)

// Checking if synmbol belongs to an Android system package
func IsSystemPackage(packageName string) bool {
	for _, p := range androidPackagePrefixes {
		if strings.HasPrefix(packageName, p) {
			return true
		}
	}
	return false
}
