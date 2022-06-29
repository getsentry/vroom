package android

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

type AndroidEvent struct {
	Action   string    `json:"action,omitempty"`
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
