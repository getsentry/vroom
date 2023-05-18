package nodetree

import (
	"hash"
	"hash/fnv"
	"strings"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
)

var (
	obfuscationSupportedPlatforms = map[platform.Platform]struct{}{
		platform.Android: {},
		platform.Java:    {},
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

		EndNS       uint64              `json:"-"`
		Frame       frame.Frame         `json:"-"`
		SampleCount int                 `json:"-"`
		StartNS     uint64              `json:"-"`
		ProfileIDs  map[string]struct{} `json:"profile_ids,omitempty"`
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
	Fingerprint   uint64   `json:"fingerprint"`
	Function      string   `json:"function"`
	Package       string   `json:"package"`
	InApp         bool     `json:"in_app"`
	SelfTimesNS   []uint64 `json:"self_times_ns"`
	SumSelfTimeNS uint64   `json:"-"`
	SampleCount   int      `json:"-"`
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
func (n *Node) CollectFunctions(profilePlatform platform.Platform, results map[uint64]CallTreeFunction) (uint64, uint64) {
	var childrenApplicationDurationNS uint64
	var childrenSystemDurationNS uint64

	// determine the amount of time spent in application vs system functions in the children
	for _, child := range n.Children {
		applicationDurationNS, systemDurationNS := child.CollectFunctions(profilePlatform, results)
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

	frameFunction := n.Frame.Function
	framePackage := n.Frame.ModuleOrPackage()

	var selfTimeNS uint64

	if shouldAggregateFrame(profilePlatform, frameFunction, framePackage) {
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
			h := fnv.New64()
			h.Write([]byte(framePackage))
			h.Write([]byte{':'})
			h.Write([]byte(frameFunction))
			fingerprint := h.Sum64()

			function, exists := results[fingerprint]
			if !exists {
				results[fingerprint] = CallTreeFunction{
					Fingerprint:   fingerprint,
					Function:      n.Frame.Function,
					Package:       framePackage,
					InApp:         n.IsApplication,
					SelfTimesNS:   []uint64{selfTimeNS},
					SumSelfTimeNS: selfTimeNS,
					SampleCount:   n.SampleCount,
				}
			} else {
				function.SelfTimesNS = append(function.SelfTimesNS, selfTimeNS)
				function.SumSelfTimeNS += selfTimeNS
				function.SampleCount += n.SampleCount
				results[fingerprint] = function
			}
		}
	}

	// this pair represents the time spent in application functions vs
	// time spent in system functions by this function and all of its descendents
	return applicationDurationNS, n.DurationNS - applicationDurationNS
}

func shouldAggregateFrame(profilePlatform platform.Platform, frameFunction string, framePackage string) bool {
	// frames with no name are not valuable for aggregation
	if frameFunction == "" {
		return false
	}

	_, obfuscationSupported := obfuscationSupportedPlatforms[profilePlatform]
	if obfuscationSupported {
		// obfuscated package names often don't contain a dot (`.`)
		if !strings.Contains(framePackage, ".") {
			return false
		}
	}

	// all other frames are safe to aggregate
	return true
}
