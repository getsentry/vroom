package nodetree

import (
	"hash"
	"strings"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/utils"
)

var (
	obfuscationSupportedPlatforms = map[platform.Platform]struct{}{
		platform.Android: {},
		platform.Java:    {},
	}

	symbolicationSupportedPlatforms = map[platform.Platform]struct{}{
		platform.JavaScript: {},
		platform.Node:       {},
		platform.Cocoa:      {},
	}

	functionDenyListByPlatform = map[platform.Platform]map[string]struct{}{
		platform.Cocoa: {
			"main": {},
		},
	}
)

type (
	Node struct {
		Children      []*Node `json:"children,omitempty"`
		DurationNS    uint64  `json:"duration_ns"`
		Fingerprint   uint64  `json:"fingerprint"`
		IsApplication bool    `json:"is_application"`
		Line          uint32  `json:"line,omitempty"`
		Name          string  `json:"name"`
		Package       string  `json:"package"`
		Path          string  `json:"path,omitempty"`

		EndNS       uint64                             `json:"-"`
		Frame       frame.Frame                        `json:"-"`
		SampleCount int                                `json:"-"`
		StartNS     uint64                             `json:"-"`
		ProfileIDs  map[string]struct{}                `json:"profile_ids,omitempty"`
		Profiles    map[utils.ExampleMetadata]struct{} `json:"profiles,omitempty"`
	}
)

func NodeFromFrame(f frame.Frame, start, end, fingerprint uint64) *Node {
	inApp := true
	if f.InApp != nil {
		inApp = *f.InApp
	}
	n := Node{
		EndNS:         end,
		Fingerprint:   fingerprint,
		Frame:         f,
		IsApplication: inApp,
		Line:          f.Line,
		Name:          f.Function,
		Package:       f.ModuleOrPackage(),
		Path:          f.Path,
		SampleCount:   1,
		StartNS:       start,
		ProfileIDs:    map[string]struct{}{},
		Profiles:      map[utils.ExampleMetadata]struct{}{},
	}
	if end > 0 {
		n.DurationNS = n.EndNS - n.StartNS
	}
	return &n
}

func (n *Node) Update(timestamp uint64) {
	n.SampleCount++
	n.SetDuration(timestamp)
}

func (n *Node) ToFrame() frame.Frame {
	n.Frame.Data.SymbolicatorStatus = n.Frame.Status
	return n.Frame
}

func (n *Node) SetDuration(t uint64) {
	n.EndNS = t
	n.DurationNS = n.EndNS - n.StartNS
}

func (n *Node) WriteToHash(h hash.Hash) {
	if n.Package == "" && n.Name == "" {
		h.Write([]byte("-"))
	} else {
		h.Write([]byte(n.Package))
		h.Write([]byte(n.Name))
	}
}

type CallTreeFunction struct {
	Fingerprint   uint32   `json:"fingerprint"`
	Function      string   `json:"function"`
	Package       string   `json:"package"`
	InApp         bool     `json:"in_app"`
	SelfTimesNS   []uint64 `json:"self_times_ns"`
	SumSelfTimeNS uint64   `json:"-"`
	SampleCount   int      `json:"-"`
	ThreadID      string   `json:"thread_id"`
	MaxDuration   uint64   `json:"-"`
}

// `CollectionFunctions` walks the node tree, collects any function with a non zero
// self-time and writes them into the `results` parameter.
//
// The meaning of self-time is slightly modified here to adapt better for our use case.
//
// For system functions, the self-time is what you would expect, it's the difference
// between the duration of the function, and the sum of the duration of it's children.
// e.g. if `foo` is a system function with a duration of 100ms, and it has 3 children
// with durations 20ms, 30ms and 40ms respectively, the self-time of `foo` will be 10ms
// because 100ms - 20ms - 30ms - 40ms = 10ms.
//
// For application functions, the self-time only looks at the time spent by it's
// descendents that are also application functions. That is, system functions do not
// affect the self-time of application functions.
// e.g. if `bar` is an application function with a duration of 100ms, and it has 3
// children with durations 20ms, 30ms, and 40ms, and they are system, application, system
// functions respectively, the self-time of `bar` will be 70ms because
// 100ms - 30ms = 70ms.
func (n *Node) CollectFunctions(
	results map[uint32]CallTreeFunction,
	threadID string,
) (uint64, uint64) {
	var childrenApplicationDurationNS uint64
	var childrenSystemDurationNS uint64

	// determine the amount of time spent in application vs system functions in the children
	for _, child := range n.Children {
		applicationDurationNS, systemDurationNS := child.CollectFunctions(results, threadID)
		childrenApplicationDurationNS += applicationDurationNS
		childrenSystemDurationNS += systemDurationNS
	}

	// calculate the time spent in application functions in this function
	applicationDurationNS := childrenApplicationDurationNS
	// in the event that the time spent in application functions in the descendents exceed
	// the frame duration, we cap it at the frame duration
	if applicationDurationNS > n.DurationNS {
		applicationDurationNS = n.DurationNS
	}

	var selfTimeNS uint64

	if shouldAggregateFrame(n.Frame) {
		if n.IsApplication {
			// cannot use `n.DurationNS - childrenApplicationDurationNS > 0` in case it underflows
			if n.DurationNS > childrenApplicationDurationNS {
				// application function's self time only looks at the time
				// spent in application function in its descendents
				selfTimeNS = n.DurationNS - childrenApplicationDurationNS

				// credit the self time of this application function
				// to the total time spent in application functions
				applicationDurationNS += selfTimeNS
			}
		} else {
			// cannot use `n.DurationNS - childrenApplicationDurationNS - childrenSystemDurationNS` in case it underflows
			if n.DurationNS > childrenApplicationDurationNS+childrenSystemDurationNS {
				// system function's self time looks at all descendents of its descendents
				selfTimeNS = n.DurationNS - childrenApplicationDurationNS - childrenSystemDurationNS
			}
		}

		if selfTimeNS > 0 {
			// casting to an uint32 here because snuba does not handle uint64 values
			// well as it is converted to a float somewhere
			// not changing to the 32 bit hash function here to preserve backwards
			// compatibility with existing fingerprints that we can cast
			fingerprint := n.Frame.Fingerprint()

			function, exists := results[fingerprint]
			if !exists {
				results[fingerprint] = CallTreeFunction{
					Fingerprint:   fingerprint,
					Function:      n.Frame.Function,
					Package:       n.Frame.ModuleOrPackage(),
					InApp:         n.IsApplication,
					SelfTimesNS:   []uint64{selfTimeNS},
					SumSelfTimeNS: selfTimeNS,
					SampleCount:   n.SampleCount,
					ThreadID:      threadID,
					MaxDuration:   selfTimeNS,
				}
			} else {
				function.SelfTimesNS = append(function.SelfTimesNS, selfTimeNS)
				function.SumSelfTimeNS += selfTimeNS
				function.SampleCount += n.SampleCount
				if selfTimeNS > function.MaxDuration {
					function.MaxDuration = selfTimeNS
					if threadID != function.ThreadID {
						function.ThreadID = threadID
					}
				}
				results[fingerprint] = function
			}
		}
	}

	// this pair represents the time spent in application functions vs
	// time spent in system functions by this function and all of its descendents
	return applicationDurationNS, n.DurationNS - applicationDurationNS
}

func shouldAggregateFrame(frame frame.Frame) bool {
	frameFunction := frame.Function

	// frames with no name are not valuable for aggregation
	if frameFunction == "" {
		return false
	}

	// hard coded list of functions that we should not aggregate by
	if functionDenyList, exists := functionDenyListByPlatform[frame.Platform]; exists {
		if _, exists = functionDenyList[frameFunction]; exists {
			return false
		}
	}

	if _, obfuscationSupported := obfuscationSupportedPlatforms[frame.Platform]; obfuscationSupported {
		/*
			There are 4 possible deobfuscation statuses
			1. deobfuscated	- The frame was successfully deobfuscated.
			2. partial			- The frame was only partially deobfuscated.
												(likely just the class name and not the method name)
			3. missing			- The frame could not be deobfuscated, not found in the mapping file.
												(likely to be a system library that should not be obfuscated)
			4. <no status>	- The frame did not go through deobfuscation. No mapping file specified.

			Only the `partial` status should not be aggregated because only having a deobfuscated
			class names makes grouping ineffective.
		*/
		if frame.Data.DeobfuscationStatus == "partial" {
			return false
		}

		// obfuscated package names often don't contain a dot (`.`)
		framePackage := frame.ModuleOrPackage()
		if !strings.Contains(framePackage, ".") {
			return false
		}
	}

	if _, symbolicationSupported := symbolicationSupportedPlatforms[frame.Platform]; symbolicationSupported {
		return isSymbolicatedFrame(frame)
	}

	// all other frames are safe to aggregate
	return true
}

func (n *Node) Close(timestamp uint64) {
	if n.EndNS == 0 {
		n.SetDuration(timestamp)
	} else {
		timestamp = n.EndNS
	}
	for _, c := range n.Children {
		c.Close(timestamp)
	}
}

func isSymbolicatedFrame(f frame.Frame) bool {
	// React-native case
	if f.Platform == platform.JavaScript && f.IsReactNative {
		if f.Data.JsSymbolicated != nil && *f.Data.JsSymbolicated {
			return true
		}
		return false
	} else if f.Platform == platform.JavaScript || f.Platform == platform.Node {
		// else, if it's not a react-native but simply a js frame from either
		// browser js or node, for now we'll simply consider everything as symbolicated
		// and just ingest into metrics
		return true
	}
	return f.Data.SymbolicatorStatus == "symbolicated"
}
