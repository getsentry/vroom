package android

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"sort"
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
	sort.SliceStable(p.Events, func(i, j int) bool {
		a, b := p.Events[i], p.Events[j]
		return a.ThreadID < b.ThreadID && buildTimestamp(a.Time) < buildTimestamp(b.Time)
	})

	nodesPerThread := make(map[uint64][]*nodetree.Node)
	enters := make(map[uint64][]AndroidEvent)
	for _, e := range p.Events {
		switch e.Action {
		case EnterAction:
			enters[e.ThreadID] = append(enters[e.ThreadID], e)
		case ExitAction, UnwindAction:
			if len(enters[e.ThreadID]) == 0 {
				continue
			}
			i := len(enters[e.ThreadID]) - 1
			ee := enters[e.ThreadID][i]
			enters[e.ThreadID] = enters[e.ThreadID][:i]

			start := buildTimestamp(ee.Time)
			end := buildTimestamp(e.Time)
			m := p.Methods[e.MethodID]
			nodesPerThread[e.ThreadID] = append(nodesPerThread[e.ThreadID], nodetree.NodeFromFrame(m.ClassName, m.Name, m.SourceFile, m.SourceLine, start, end, m.ID))
		}
	}

	trees := make(map[uint64][]*nodetree.Node)
	for threadID, nodes := range nodesPerThread {
		root := nodetree.NodeFromFrame("root", "root", "", 0, 0, math.MaxUint64, 0)
		fingerprint := fnv.New64()
		buffer := make([]byte, 8)
		binary.LittleEndian.PutUint64(buffer, threadID)
		fingerprint.Write(buffer)
		for _, n := range nodes {
			root.Insert(n, fingerprint)
		}
		trees[threadID] = root.Children
	}

	return trees
}
