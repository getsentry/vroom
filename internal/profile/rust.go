package profile

import (
	"strings"
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
