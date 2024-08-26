package chunk

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"

	"github.com/getsentry/vroom/internal/clientsdk"
	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/utils"
)

var (
	ErrInvalidStackID = errors.New("profile contains invalid stack id")
	ErrInvalidFrameID = errors.New("profile contains invalid frame id")
)

type (
	// Chunk is an implementation of the Sample V2 format.
	Chunk struct {
		ID         string `json:"chunk_id"`
		ProfilerID string `json:"profiler_id"`

		DebugMeta debugmeta.DebugMeta `json:"debug_meta"`

		ClientSDK   clientsdk.ClientSDK `json:"client_sdk"`
		Environment string              `json:"environment"`
		Platform    platform.Platform   `json:"platform"`
		Release     string              `json:"release"`

		Version string `json:"version"`

		Profile Data `json:"profile"`

		OrganizationID uint64  `json:"organization_id"`
		ProjectID      uint64  `json:"project_id"`
		Received       float64 `json:"received"`
		RetentionDays  int     `json:"retention_days"`

		Measurements json.RawMessage `json:"measurements"`

		Options utils.Options `json:"options,omitempty"`
	}

	Data struct {
		Frames         []frame.Frame                    `json:"frames"`
		Samples        []Sample                         `json:"samples"`
		Stacks         [][]int                          `json:"stacks"`
		ThreadMetadata map[string]sample.ThreadMetadata `json:"thread_metadata"`
	}

	Sample struct {
		StackID   int     `json:"stack_id"`
		ThreadID  string  `json:"thread_id"`
		Timestamp float64 `json:"timestamp"`
	}
)

func (c *Chunk) StoragePath() string {
	return StoragePath(
		c.OrganizationID,
		c.ProjectID,
		c.ProfilerID,
		c.ID,
	)
}

func (c *Chunk) StartEndTimestamps() (float64, float64) {
	count := len(c.Profile.Samples)
	if count == 0 {
		return 0, 0
	}
	return c.Profile.Samples[0].Timestamp, c.Profile.Samples[count-1].Timestamp
}

func (c *Chunk) Normalize() {
	for i := range c.Profile.Frames {
		f := c.Profile.Frames[i]
		f.Normalize(c.Platform)
		c.Profile.Frames[i] = f
	}

	if c.Platform == platform.Python {
		c.Profile.trimPythonStacks()
	}
}

func StoragePath(OrganizationID uint64, ProjectID uint64, ProfilerID string, ID string) string {
	return fmt.Sprintf(
		"%d/%d/%s/%s",
		OrganizationID,
		ProjectID,
		ProfilerID,
		ID,
	)
}

// CallTrees generates call trees from samples.
func (c Chunk) CallTrees(activeThreadID *string) (map[string][]*nodetree.Node, error) {
	sort.SliceStable(c.Profile.Samples, func(i, j int) bool {
		return c.Profile.Samples[i].Timestamp < c.Profile.Samples[j].Timestamp
	})

	treesByThreadID := make(map[string][]*nodetree.Node)
	samplesByThreadID := make(map[string][]Sample)

	for _, s := range c.Profile.Samples {
		samplesByThreadID[s.ThreadID] = append(samplesByThreadID[s.ThreadID], s)
	}

	var current *nodetree.Node
	h := fnv.New64()
	for _, samples := range samplesByThreadID {
		// The last sample is not represented, only used for its timestamp.
		for sampleIndex := 0; sampleIndex < len(samples)-1; sampleIndex++ {
			s := samples[sampleIndex]
			if activeThreadID != nil && s.ThreadID != *activeThreadID {
				continue
			}

			if len(c.Profile.Stacks) <= s.StackID {
				return nil, ErrInvalidStackID
			}

			stack := c.Profile.Stacks[s.StackID]
			for i := len(stack) - 1; i >= 0; i-- {
				if len(c.Profile.Frames) <= stack[i] {
					return nil, ErrInvalidFrameID
				}
			}

			// here while we save the nextTimestamp val, we convert it to nanosecond
			// since the Node struct and utilities use uint64 ns values
			nextTimestamp := uint64(samples[sampleIndex+1].Timestamp * 1e9)
			sampleTimestamp := uint64(s.Timestamp * 1e9)

			for i := len(stack) - 1; i >= 0; i-- {
				f := c.Profile.Frames[stack[i]]
				f.WriteToHash(h)
				fingerprint := h.Sum64()
				if current == nil {
					i := len(treesByThreadID[s.ThreadID]) - 1
					if i >= 0 && treesByThreadID[s.ThreadID][i].Fingerprint == fingerprint &&
						treesByThreadID[s.ThreadID][i].EndNS == sampleTimestamp {
						current = treesByThreadID[s.ThreadID][i]
						current.Update(nextTimestamp)
					} else {
						n := nodetree.NodeFromFrame(f, sampleTimestamp, nextTimestamp, fingerprint)
						treesByThreadID[s.ThreadID] = append(treesByThreadID[s.ThreadID], n)
						current = n
					}
				} else {
					i := len(current.Children) - 1
					if i >= 0 && current.Children[i].Fingerprint == fingerprint && current.Children[i].EndNS == sampleTimestamp {
						current = current.Children[i]
						current.Update(nextTimestamp)
					} else {
						n := nodetree.NodeFromFrame(f, sampleTimestamp, nextTimestamp, fingerprint)
						current.Children = append(current.Children, n)
						current = n
					}
				}
			} // end stack loop
			h.Reset()
			current = nil
		}
	}

	return treesByThreadID, nil
}

func (d *Data) trimPythonStacks() {
	// Find the module frame index in frames
	mfi := -1
	for i, f := range d.Frames {
		if f.File == "<string>" && f.Function == "<module>" {
			mfi = i
		}
	}

	// We do nothing if we don't find it
	if mfi == -1 {
		return
	}

	for si, s := range d.Stacks {
		l := len(s)

		// ignore empty stacks
		if l == 0 {
			continue
		}

		// found the module frame so trim it
		if s[l-1] == mfi {
			d.Stacks[si] = d.Stacks[si][:l-1]
		}
	}
}
