package chunk

import (
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/getsentry/vroom/internal/clientsdk"
	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/utils"
)

type (
	AndroidChunk struct {
		BuildID    string `json:"build_id,omitempty"`
		ID         string `json:"chunk_id"`
		ProfilerID string `json:"profiler_id"`

		DebugMeta debugmeta.DebugMeta `json:"debug_meta"`

		ClientSDK   clientsdk.ClientSDK `json:"client_sdk"`
		DurationNS  uint64              `json:"duration_ns"`
		Environment string              `json:"environment"`
		Platform    platform.Platform   `json:"platform"`
		Release     string              `json:"release"`
		Timestamp   float64             `json:"timestamp"`

		Profile      profile.Android `json:"profile"`
		Measurements json.RawMessage `json:"measurements"`

		OrganizationID uint64  `json:"organization_id"`
		ProjectID      uint64  `json:"project_id"`
		Received       float64 `json:"received"`
		RetentionDays  int     `json:"retention_days"`

		Options utils.Options `json:"options,omitempty"`
	}
)

func (c AndroidChunk) StoragePath() string {
	return StoragePath(
		c.OrganizationID,
		c.ProjectID,
		c.ProfilerID,
		c.ID,
	)
}

func (c AndroidChunk) DurationMS() uint64 {
	return uint64(time.Duration(c.DurationNS).Milliseconds())
}

func (c AndroidChunk) CallTrees(activeThreadID *string) (map[string][]*nodetree.Node, error) {
	return c.CallTreesWithMaxDepth(activeThreadID, profile.MaxStackDepth)
}

func (c AndroidChunk) CallTreesWithMaxDepth(activeThreadID *string, maxDepth int) (map[string][]*nodetree.Node, error) {
	p := c.Profile
	// in case wall-clock.secs is not monotonic, "fix" it
	p.FixSamplesTime()

	buildTimestamp := p.TimestampGetter()
	treesByThreadID := make(map[string][]*nodetree.Node)
	stacks := make(map[uint64][]*nodetree.Node)
	stackDepth := make(map[uint64]int)

	methods := make(map[uint64]profile.AndroidMethod)
	for _, m := range p.Methods {
		methods[m.ID] = m
	}

	closeFrame := func(threadID uint64, ts uint64, i int) {
		n := stacks[threadID][i]
		n.Update(ts)
		n.SampleCount = int(math.Ceil(float64(n.DurationNS) / float64((10 * time.Millisecond))))
	}

	var maxTimestampNS uint64
	enterPerMethod := make(map[uint64]int)
	exitPerMethod := make(map[uint64]int)

	for _, e := range p.Events {
		tid := strconv.FormatUint(e.ThreadID, 10)
		if activeThreadID != nil && tid != *activeThreadID {
			continue
		}

		ts := buildTimestamp(e.Time)
		if ts > maxTimestampNS {
			maxTimestampNS = ts
		}

		switch e.Action {
		case profile.EnterAction:
			m, exists := methods[e.MethodID]
			if !exists {
				methods[e.MethodID] = profile.AndroidMethod{
					ClassName: "unknown",
					ID:        e.MethodID,
					Name:      "unknown",
				}
			}
			stackDepth[e.ThreadID]++
			if stackDepth[e.ThreadID] > maxDepth {
				continue
			}
			enterPerMethod[e.MethodID]++
			n := nodetree.NodeFromFrame(m.Frame(), ts, 0, 0)
			if len(stacks[e.ThreadID]) == 0 {
				treesByThreadID[tid] = append(treesByThreadID[tid], n)
			} else {
				i := len(stacks[e.ThreadID]) - 1
				stacks[e.ThreadID][i].Children = append(stacks[e.ThreadID][i].Children, n)
			}
			stacks[e.ThreadID] = append(stacks[e.ThreadID], n)
			n.Fingerprint = profile.GenerateFingerprint(stacks[e.ThreadID])
		case profile.ExitAction, profile.UnwindAction:
			stackDepth[e.ThreadID]--
			if stackDepth[e.ThreadID] > maxDepth {
				continue
			}
			if len(stacks[e.ThreadID]) == 0 {
				continue
			}
			i := len(stacks[e.ThreadID]) - 1
			var eventSkipped bool
			for ; i >= 0; i-- {
				n := stacks[e.ThreadID][i]
				if n.Frame.MethodID != e.MethodID &&
					enterPerMethod[e.MethodID] <= exitPerMethod[e.MethodID] {
					eventSkipped = true
					break
				}
				closeFrame(e.ThreadID, ts, i)
				exitPerMethod[e.MethodID]++
				if n.Frame.MethodID == e.MethodID {
					break
				}
			}
			// If we didn't skip the event, we should cut the stack accordingly.
			if !eventSkipped {
				stacks[e.ThreadID] = stacks[e.ThreadID][:i]
			}
		}
	}

	// Close remaining open frames.
	for threadID, stack := range stacks {
		for i := len(stack) - 1; i >= 0; i-- {
			closeFrame(threadID, maxTimestampNS, i)
		}
	}
	for _, trees := range treesByThreadID {
		for _, root := range trees {
			root.Close(maxTimestampNS)
		}
	}

	return treesByThreadID, nil
}

func (c AndroidChunk) SDKName() string {
	return c.ClientSDK.Name
}

func (c AndroidChunk) SDKVersion() string {
	return c.ClientSDK.Version
}

func (c AndroidChunk) EndTimestamp() float64 {
	return c.Timestamp + float64(c.DurationNS)*1e-9
}

func (c AndroidChunk) GetEnvironment() string {
	return c.Environment
}

func (c AndroidChunk) GetID() string {
	return c.ID
}

func (c AndroidChunk) GetPlatform() platform.Platform {
	return c.Platform
}

func (c AndroidChunk) GetProfilerID() string {
	return c.ProfilerID
}

func (c AndroidChunk) GetProjectID() uint64 {
	return c.ProjectID
}

func (c AndroidChunk) GetReceived() float64 {
	return c.Received
}

func (c AndroidChunk) GetRelease() string {
	return c.Release
}

func (c AndroidChunk) GetRetentionDays() int {
	return c.RetentionDays
}

func (c AndroidChunk) StartTimestamp() float64 {
	return c.Timestamp
}

func (c AndroidChunk) GetOrganizationID() uint64 {
	return c.OrganizationID
}

func (c AndroidChunk) GetOptions() utils.Options {
	return c.Options
}

func (c AndroidChunk) GetFrameWithFingerprint(target uint32) (frame.Frame, error) {
	for _, m := range c.Profile.Methods {
		f := m.Frame()
		if f.Fingerprint() == target {
			return f, nil
		}
	}
	return frame.Frame{}, frame.ErrFrameNotFound
}

func (c *AndroidChunk) Normalize() {
}
