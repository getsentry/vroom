package aggregate

import (
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"sort"
	"strings"

	"github.com/getsentry/vroom/internal/calltree"
	"github.com/getsentry/vroom/internal/quantile"
)

type (
	DisplayModeType int
)

const (
	DisplayModeIOS DisplayModeType = iota
	DisplayModeAndroid
)

func newCallTreeFrameP(root *calltree.AggregateCallTree, hashOfParents []byte, displayMode DisplayModeType) Frame {
	// Compute the hash of the current frame, then append the hash of the parents
	// so that we get the hash of all of the nodes on the path to this node in the tree.
	h := computeFunctionHash(root.Image, root.Symbol)
	h.Write(hashOfParents)
	currentHash := h.Sum(nil)

	children := make([]Frame, 0, len(root.Children))
	for _, child := range root.Children {
		// Now further sub-compute the hashes of its children.
		children = append(children, newCallTreeFrameP(child, currentHash, displayMode))
	}

	var image, symbol string
	var isApplicationSymbol bool

	switch displayMode {
	case DisplayModeIOS:
		image = root.Image
		symbol = root.Symbol
		isApplicationSymbol = IsIOSApplicationImage(root.Path)
	case DisplayModeAndroid:
		image = root.Image
		symbol = root.Symbol
		isApplicationSymbol = !IsAndroidSystemPackage(root.Image)
	default:
		image = root.Image
		symbol = root.Symbol
	}

	return Frame{
		Children:              children,
		ID:                    fmt.Sprintf("%x", currentHash),
		Image:                 image,
		IsApplicationSymbol:   isApplicationSymbol,
		Line:                  root.Line,
		Path:                  root.Path,
		SelfDurationNs:        quantileToAggQuantiles(quantile.Quantile{Xs: root.SelfDurationsNs}),
		SelfDurationNsValues:  root.SelfDurationsNs,
		Symbol:                symbol,
		TotalDurationNs:       quantileToAggQuantiles(quantile.Quantile{Xs: root.TotalDurationsNs}),
		TotalDurationNsValues: root.TotalDurationsNs,
	}
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
func IsAndroidSystemPackage(packageName string) bool {
	for _, p := range androidPackagePrefixes {
		if strings.HasPrefix(packageName, p) {
			return true
		}
	}
	return false
}

// isApplicationSymbol determines whether the image represents that of the application
// binary (or a binary embedded in the application binary) by checking its path.
func IsIOSApplicationImage(image string) bool {
	// These are the path patterns that iOS uses for applications, system
	// libraries are stored elsewhere.
	//
	// Must be kept in sync with the corresponding Python implementation of
	// this function in python/symbolicate/__init__.py
	return strings.HasPrefix(image, "/private/var/containers") ||
		strings.HasPrefix(image, "/var/containers") ||
		strings.Contains(image, "/Developer/Xcode/DerivedData") ||
		strings.Contains(image, "/data/Containers/Bundle/Application")
}

func computeFunctionHash(image, symbol string) hash.Hash {
	h := md5.New()
	_, _ = io.WriteString(h, image)
	_, _ = io.WriteString(h, symbol)
	return h
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Structs

/** A container type to group aggregations by app_version, interaction and
 * entry_type. **/
type AggregationResult struct {
	AppVersion      string    `json:"app_version,omitempty"`
	TransactionName string    `json:"interaction,omitempty"`
	EntityType      string    `json:"entry_type,omitempty"`
	RowCount        uint32    `json:"row_count"`
	Aggregation     Aggregate `json:"aggregation"`
}

type Aggregate struct {
	FunctionCalls       []FunctionCall        `json:"function_call"`
	FunctionToCallTrees map[string][]CallTree `json:"function_to_call_trees"`
}

type FunctionCall struct {
	// The name of the binary/library that the function is in
	Image string `json:"image"`
	// String representation of the function name
	Symbol string `json:"symbol"`
	// Wall time duration for the execution of the function
	DurationNs       Quantiles `json:"duration_ns"`
	DurationNsValues []float64 `json:"-"`

	// How frequently the function is called within a single transaction
	Frequency       Quantiles `json:"frequency"`
	FrequencyValues []float64 `json:"-"`

	// Percentage of how frequently the function is called on the main thread,
	// from [0, 1]
	MainThreadPercent float32 `json:"main_thread_percent"`
	// Map from thread name to the % of the time that the function is called
	// on that thread, from [0, 1], includes the main thread.

	ThreadNameToPercent map[string]float32 `json:"thread_name_to_percent"`
	// Line is the line number for the function in its original source file,
	// if that information is available, otherwise 0.
	Line int `json:"line"`

	// Path is the path to the original source file that contains the function,
	// if that information is available, otherwise "".
	Path string `json:"path"`

	// ProfileIDs is a unique list of the profile identifiers that this function
	// appears in.
	ProfileIDs []string `json:"profile_ids"`

	ProfileIDToThreadID map[string]uint64 `json:"profile_id_to_thread_id"`

	// The key that can be used to look up this function in the
	// function_to_call_trees map.
	Key string `json:"key"`

	// List of unique transaction names where this function is found
	TransactionNames []string `json:"transaction_names"`
}

type CallTree struct {
	// An identifier that uniquely identifies the pattern for this call tree,
	// which is computed as an MD5 hash over the image & symbol for each of
	// the frames, recursively.
	ID string `json:"id"`

	// The number of times this call tree pattern was recorded.
	Count uint64 `json:"count"`

	// Map from thread name to the number of times this call tree pattern was
	// recorded for that thread.
	ThreadNameToCount map[string]uint64 `json:"thread_name_to_count"`

	// A unique list of the trace identifiers that this call tree appears in.
	ProfileIDs []string `json:"profile_ids"`

	// The root frame of this call tree.
	RootFrame Frame `json:"root_frame"`
}

type Frame struct {
	// A stable identifier that uniquely identifies this frame within the
	// tree. This identifier is an MD5 hash of image/symbol of this node and
	// all of its parents nodes, so even when the frame for a function appears
	// multiple times in the tree, each node will have a unique ID.
	ID string `json:"id"`

	// The name of the binary/library that the function is in.
	Image string `json:"image"`

	// String representation of the function name.
	Symbol string `json:"symbol"`

	// Whether the symbol exists in application code (as opposed to system/SDK
	// code)
	IsApplicationSymbol bool `json:"is_application_symbol"`

	// Line is the line number for the function in its original source file,
	// if that information is available, otherwise 0.
	Line uint32 `json:"line"`

	// Path is the path to the original source file that contains the
	// function, if that information is available, otherwise "".
	Path string `json:"-"`

	// Wall time duration for the execution of the function and its children.
	TotalDurationNs       Quantiles `json:"total_duration_ns"`
	TotalDurationNsValues []float64 `json:"-"`

	// Child frames of this frame.
	Children []Frame `json:"children"`

	// Wall time duration for the execution of the function, excluding its
	// children.
	SelfDurationNs       Quantiles `json:"self_duration_ns"`
	SelfDurationNsValues []float64 `json:"-"`
}

func (f Frame) Identifier() string {
	return fmt.Sprintf("%s:%s", calltree.ImageBaseName(f.Image), f.Symbol)
}

type Quantiles struct {
	P50 float64 `json:"p50"`
	P75 float64 `json:"p75"`
	P90 float64 `json:"p90"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

// Ios & Android Profiles

type IosFrame struct {
	AbsPath         string `json:"abs_path,omitempty"`
	Filename        string `json:"filename,omitempty"`
	Function        string `json:"function,omitempty"`
	InstructionAddr string `json:"instruction_addr,omitempty"`
	Lang            string `json:"lang,omitempty"`
	LineNo          int    `json:"lineno,omitempty"`
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
				if q, qExists := qm[tq]; qExists {
					if q < frameCount {
						qm[tq] = frameCount
					}
				} else {
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

type Symbol struct {
	Image    string `json:"image"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Filename string `json:"filename"`
	Line     int    `json:"line"`
}

// Path() returns (line, path, ok) where ok indicates whether the
// values are valid and can be used.
func (s *Symbol) GetPath() (int, string, bool) {
	if s.Filename != "" && s.Filename != "<compiler-generated>" {
		return s.Line, s.Path, true
	}
	return 0, "", false
}

func quantileToAggQuantiles(q quantile.Quantile) Quantiles {
	return Quantiles{
		P50: q.Percentile(0.5),
		P75: q.Percentile(0.75),
		P90: q.Percentile(0.90),
		P95: q.Percentile(0.95),
		P99: q.Percentile(0.99),
	}
}
