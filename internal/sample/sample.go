package sample

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/measurements"
	"github.com/getsentry/vroom/internal/metadata"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/transaction"
)

type (
	Device struct {
		Architecture   string `json:"architecture"`
		Classification string `json:"classification"`
		Locale         string `json:"locale"`
		Manufacturer   string `json:"manufacturer"`
		Model          string `json:"model"`
	}

	OS struct {
		BuildNumber string `json:"build_number"`
		Name        string `json:"name"`
		Version     string `json:"version"`
	}

	Runtime struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	Transaction struct {
		ActiveThreadID  uint64 `json:"active_thread_id"`
		ID              string `json:"id"`
		Name            string `json:"name"`
		RelativeEndNS   uint64 `json:"relative_end_ns"`
		RelativeStartNS uint64 `json:"relative_start_ns"`
		TraceID         string `json:"trace_id"`
	}

	Sample struct {
		ElapsedSinceStartNS uint64 `json:"elapsed_since_start_ns"`
		QueueAddress        string `json:"queue_address,omitempty"`
		StackID             int    `json:"stack_id"`
		State               string `json:"state"`
		ThreadID            uint64 `json:"thread_id"`
	}

	ThreadMetadata struct {
		Name     string `json:"name,omitempty"`
		Priority int    `json:"priority,omitempty"`
	}

	QueueMetadata struct {
		Label string `json:"label"`
	}

	Stack []int

	Trace struct {
		Frames         []frame.Frame             `json:"frames"`
		QueueMetadata  map[string]QueueMetadata  `json:"queue_metadata"`
		Samples        []Sample                  `json:"samples"`
		Stacks         []Stack                   `json:"stacks"`
		ThreadMetadata map[string]ThreadMetadata `json:"thread_metadata"`
	}

	SampleProfile struct {
		DebugMeta      debugmeta.DebugMeta                 `json:"debug_meta"`
		Device         Device                              `json:"device"`
		Environment    string                              `json:"environment,omitempty"`
		EventID        string                              `json:"event_id"`
		Measurements   map[string]measurements.Measurement `json:"measurements,omitempty"`
		OS             OS                                  `json:"os"`
		OrganizationID uint64                              `json:"organization_id"`
		Platform       platform.Platform                   `json:"platform"`
		ProjectID      uint64                              `json:"project_id"`
		Received       time.Time                           `json:"received"`
		Release        string                              `json:"release"`
		RetentionDays  int                                 `json:"retention_days"`
		Runtime        Runtime                             `json:"runtime"`
		Timestamp      time.Time                           `json:"timestamp"`
		Trace          Trace                               `json:"profile"`
		Transaction    Transaction                         `json:"transaction"`
		Transactions   []Transaction                       `json:"transactions"`
		Version        string                              `json:"version"`
	}
)

func (p SampleProfile) GetRelease() string {
	return p.Release
}

func (q QueueMetadata) LabeledAsMainThread() bool {
	return q.Label == "com.apple.main-thread"
}

func (t Transaction) DurationNS() uint64 {
	return t.RelativeEndNS - t.RelativeStartNS
}

func (p SampleProfile) GetOrganizationID() uint64 {
	return p.OrganizationID
}

func (p SampleProfile) GetProjectID() uint64 {
	return p.ProjectID
}

func (p SampleProfile) GetID() string {
	return p.EventID
}

func StoragePath(organizationID, projectID uint64, profileID string) string {
	return fmt.Sprintf("%d/%d/%s", organizationID, projectID, strings.Replace(profileID, "-", "", -1))
}

func (p SampleProfile) StoragePath() string {
	return StoragePath(p.OrganizationID, p.ProjectID, p.EventID)
}

func (p SampleProfile) GetPlatform() platform.Platform {
	return p.Platform
}

func (p SampleProfile) GetEnvironment() string {
	return p.Environment
}

func (p SampleProfile) GetTransaction() transaction.Transaction {
	return transaction.Transaction{
		ActiveThreadID: p.Transaction.ActiveThreadID,
		DurationNS:     p.Transaction.DurationNS(),
		ID:             p.Transaction.ID,
		Name:           p.Transaction.Name,
		TraceID:        p.Transaction.TraceID,
	}
}

func (p SampleProfile) GetTimestamp() time.Time {
	return p.Timestamp
}

func (p SampleProfile) GetReceived() time.Time {
	return p.Received
}

func (p SampleProfile) GetRetentionDays() int {
	return p.RetentionDays
}

func (p SampleProfile) GetDurationNS() uint64 {
	t := p.Transactions[0]
	return t.RelativeEndNS - t.RelativeStartNS
}

func (p SampleProfile) CallTrees() (map[uint64][]*nodetree.Node, error) {
	sort.SliceStable(p.Trace.Samples, func(i, j int) bool {
		return p.Trace.Samples[i].ElapsedSinceStartNS < p.Trace.Samples[j].ElapsedSinceStartNS
	})

	activeThreadID := p.Transactions[0].ActiveThreadID
	treesByThreadID := make(map[uint64][]*nodetree.Node)
	previousTimestamp := make(map[uint64]uint64)

	var current *nodetree.Node
	h := fnv.New64()
	for _, s := range p.Trace.Samples {
		if s.ThreadID != activeThreadID {
			continue
		}

		stack := p.Trace.Stacks[s.StackID]
		for i := len(stack) - 1; i >= 0; i-- {
			f := p.Trace.Frames[stack[i]]
			f.WriteToHash(h)
			fingerprint := h.Sum64()
			if current == nil {
				i := len(treesByThreadID[s.ThreadID]) - 1
				if i >= 0 && treesByThreadID[s.ThreadID][i].Fingerprint == fingerprint && treesByThreadID[s.ThreadID][i].EndNS == previousTimestamp[s.ThreadID] {
					current = treesByThreadID[s.ThreadID][i]
					current.Update(s.ElapsedSinceStartNS)
				} else {
					n := nodetree.NodeFromFrame(f.PackageBaseName(), f.Function, f.Path, f.Line, previousTimestamp[s.ThreadID], s.ElapsedSinceStartNS, fingerprint, p.IsApplicationFrame(f))
					treesByThreadID[s.ThreadID] = append(treesByThreadID[s.ThreadID], n)
					current = n
				}
			} else {
				i := len(current.Children) - 1
				if i >= 0 && current.Children[i].Fingerprint == fingerprint && current.Children[i].EndNS == previousTimestamp[s.ThreadID] {
					current = current.Children[i]
					current.Update(s.ElapsedSinceStartNS)
				} else {
					n := nodetree.NodeFromFrame(f.PackageBaseName(), f.Function, f.Path, f.Line, previousTimestamp[s.ThreadID], s.ElapsedSinceStartNS, fingerprint, p.IsApplicationFrame(f))
					current.Children = append(current.Children, n)
					current = n
				}
			}
		}
		h.Reset()
		previousTimestamp[s.ThreadID] = s.ElapsedSinceStartNS
		current = nil
	}

	return treesByThreadID, nil
}

// ThreadName returns the proper name of a thread.
// In all cases but cocoa, we'll have a thread name in the thread metadata and we should return that.
// In the cocoa case, we need to look at queue metadata and return that.
// Sometimes, several threads refer to the queue labeled "com.apple.main-thread" even if they're not the main thread.
// In this case, we want to only return "com.apple.main-thread" for the main thread and blank for the rest.
func (t *Trace) ThreadName(threadID, queueAddress string, mainThread bool) string {
	if m, exists := t.ThreadMetadata[threadID]; exists && m.Name != "" {
		return m.Name
	}
	if m, exists := t.QueueMetadata[queueAddress]; exists && ((m.LabeledAsMainThread() && mainThread) || !m.LabeledAsMainThread()) {
		return m.Label
	}
	return ""
}

func (p *SampleProfile) Speedscope() (speedscope.Output, error) {
	sort.SliceStable(p.Trace.Samples, func(i, j int) bool {
		return p.Trace.Samples[i].ElapsedSinceStartNS < p.Trace.Samples[j].ElapsedSinceStartNS
	})

	threadIDToProfile := make(map[uint64]*speedscope.SampledProfile)
	addressToFrameIndex := make(map[string]int)
	threadIDToPreviousTimestampNS := make(map[uint64]uint64)
	frames := make([]speedscope.Frame, 0)
	// we need to find the frame index of the main function so we can remove the frames before it
	mainFunctionFrameIndex := -1
	mainThreadID := p.Transactions[0].ActiveThreadID
	for _, sample := range p.Trace.Samples {
		threadID := strconv.FormatUint(sample.ThreadID, 10)
		stack := p.Trace.Stacks[sample.StackID]
		speedscopeProfile, exists := threadIDToProfile[sample.ThreadID]
		if !exists {
			isMainThread := sample.ThreadID == mainThreadID
			speedscopeProfile = &speedscope.SampledProfile{
				IsMainThread: isMainThread,
				Name:         p.Trace.ThreadName(threadID, sample.QueueAddress, isMainThread),
				StartValue:   sample.ElapsedSinceStartNS,
				ThreadID:     sample.ThreadID,
				Type:         speedscope.ProfileTypeSampled,
				Unit:         speedscope.ValueUnitNanoseconds,
			}
			if metadata, exists := p.Trace.ThreadMetadata[threadID]; exists {
				speedscopeProfile.Priority = metadata.Priority
			}
			threadIDToProfile[sample.ThreadID] = speedscopeProfile
		} else {
			speedscopeProfile.Weights = append(speedscopeProfile.Weights, sample.ElapsedSinceStartNS-threadIDToPreviousTimestampNS[sample.ThreadID])
		}

		speedscopeProfile.EndValue = sample.ElapsedSinceStartNS
		threadIDToPreviousTimestampNS[sample.ThreadID] = sample.ElapsedSinceStartNS

		samp := make([]int, 0, len(stack))
		for i := len(stack) - 1; i >= 0; i-- {
			fr := p.Trace.Frames[stack[i]]
			address := fr.ID()
			frameIndex, ok := addressToFrameIndex[address]
			if !ok {
				frameIndex = len(frames)
				symbolName := fr.Function
				if symbolName == "" {
					symbolName = fmt.Sprintf("unknown (%s)", address)
				} else if mainFunctionFrameIndex == -1 {
					if isMainFrame, i := fr.IsMain(); isMainFrame {
						mainFunctionFrameIndex = frameIndex + i
					}
				}
				addressToFrameIndex[address] = frameIndex
				frames = append(frames, speedscope.Frame{
					Col:           fr.Column,
					File:          fr.File,
					Image:         fr.PackageBaseName(),
					Inline:        fr.IsInline(),
					IsApplication: p.IsApplicationFrame(fr),
					Line:          fr.Line,
					Name:          symbolName,
					Path:          fr.Path,
				})
			}
			samp = append(samp, frameIndex)
		}
		speedscopeProfile.Samples = append(speedscopeProfile.Samples, samp)
	} // end loop speedscope.SampledProfiles
	var mainThreadProfileIndex int
	allProfiles := make([]interface{}, 0)
	for _, prof := range threadIDToProfile {
		if prof.IsMainThread {
			mainThreadProfileIndex = len(allProfiles)
		}
		prof.Weights = append(prof.Weights, 0)
		allProfiles = append(allProfiles, prof)
	}

	return speedscope.Output{
		ActiveProfileIndex: mainThreadProfileIndex,
		DurationNS:         p.Transactions[0].DurationNS(),
		Images:             p.DebugMeta.Images,
		Metadata: speedscope.ProfileMetadata{
			ProfileView: speedscope.ProfileView{
				Architecture:         p.Device.Architecture,
				DeviceClassification: p.Device.Classification,
				DeviceLocale:         p.Device.Locale,
				DeviceManufacturer:   p.Device.Manufacturer,
				DeviceModel:          p.Device.Model,
				DeviceOSName:         p.OS.Name,
				DeviceOSVersion:      p.OS.Version,
				DurationNS:           p.Transactions[0].DurationNS(),
				Environment:          p.Environment,
				OrganizationID:       p.OrganizationID,
				Platform:             p.Platform,
				ProfileID:            p.EventID,
				ProjectID:            p.ProjectID,
				Received:             p.Timestamp,
				TraceID:              p.Transactions[0].TraceID,
				TransactionID:        p.Transactions[0].ID,
				TransactionName:      p.Transactions[0].Name,
			},
			Version: p.Release,
		},
		Platform:        p.Platform,
		ProfileID:       p.EventID,
		Profiles:        allProfiles,
		ProjectID:       p.ProjectID,
		Shared:          speedscope.SharedData{Frames: frames},
		TransactionName: p.Transactions[0].Name,
		Version:         p.Release,
		Measurements:    p.Measurements,
	}, nil
}

func (p *SampleProfile) IsApplicationFrame(f frame.Frame) bool {
	if f.InApp != nil {
		return *f.InApp
	}
	switch p.Platform {
	case "node":
		return f.IsNodeApplicationFrame()
	case "cocoa":
		return f.IsIOSApplicationFrame()
	case "rust":
		return f.IsRustApplicationFrame()
	case "python":
		return f.IsPythonApplicationFrame()
	}
	return true
}

func (p *SampleProfile) Metadata() metadata.Metadata {
	return metadata.Metadata{
		Architecture:         p.Device.Architecture,
		DeviceClassification: p.Device.Classification,
		DeviceLocale:         p.Device.Locale,
		DeviceManufacturer:   p.Device.Manufacturer,
		DeviceModel:          p.Device.Model,
		DeviceOSBuildNumber:  p.OS.BuildNumber,
		DeviceOSName:         p.OS.Name,
		DeviceOSVersion:      p.OS.Version,
		ID:                   p.EventID,
		ProjectID:            strconv.FormatUint(p.ProjectID, 10),
		Timestamp:            p.Received.Unix(),
		TraceDurationMs:      float64(p.Transactions[0].DurationNS()) / 1_000_000,
		TransactionID:        p.Transactions[0].ID,
		TransactionName:      p.Transactions[0].Name,
		VersionName:          p.Release,
	}
}

func (p *SampleProfile) Raw() []byte {
	return []byte{}
}

func (p *SampleProfile) ReplaceIdleStacks() {
	p.Trace.ReplaceIdleStacks()
}

func (t Trace) SamplesByThreadD() ([]uint64, map[uint64][]*Sample) {
	samples := make(map[uint64][]*Sample)
	var threadIDs []uint64
	for i, s := range t.Samples {
		if _, exists := samples[s.ThreadID]; !exists {
			threadIDs = append(threadIDs, s.ThreadID)
		}
		samples[s.ThreadID] = append(samples[s.ThreadID], &t.Samples[i])
	}
	sort.SliceStable(threadIDs, func(i, j int) bool {
		return threadIDs[i] < threadIDs[j]
	})
	return threadIDs, samples
}

func (p *Trace) ReplaceIdleStacks() {
	threadIDs, samplesByThreadID := p.SamplesByThreadD()

	for _, threadID := range threadIDs {
		samples := samplesByThreadID[threadID]
		previousActiveStackID := -1
		var nextActiveSampleIndex, nextActiveStackID int

		for i := 0; i < len(samples); i++ {
			s := samples[i]

			// keep track of the previous active sample as we go
			if p.Stacks[s.StackID].IsActive() {
				previousActiveStackID = s.StackID
				continue
			}

			// if there's no frame, the thread is considered idle at this time
			s.State = "idle"

			// if it's an idle stack but we don't have a previous active stack
			// we keep looking
			if previousActiveStackID == -1 {
				continue
			}

			if i >= nextActiveSampleIndex {
				nextActiveSampleIndex, nextActiveStackID = p.findNextActiveStackID(samples, i)
				if nextActiveSampleIndex == -1 {
					// no more active sample on this thread
					for ; i < len(samples); i++ {
						samples[i].State = "idle"
					}
					break
				}
			}

			previousFrames := p.framesList(previousActiveStackID)
			nextFrames := p.framesList(nextActiveStackID)
			commonFrames := findCommonFrames(previousFrames, nextFrames)

			// add the common stack to the list of stacks
			commonStack := make([]int, 0, len(commonFrames))
			for _, frame := range commonFrames {
				commonStack = append(commonStack, frame.index)
			}
			commonStackID := len(p.Stacks)
			p.Stacks = append(p.Stacks, commonStack)

			// replace all idle stacks until next active sample
			for ; i < nextActiveSampleIndex; i++ {
				samples[i].StackID = commonStackID
				samples[i].State = "idle"
			}
		}
	}
}

type frameTuple struct {
	index int
	frame frame.Frame
}

func (t Trace) framesList(stackID int) []frameTuple {
	stack := t.Stacks[stackID]
	frames := make([]frameTuple, 0, len(stack))
	for _, frameID := range stack {
		frames = append(frames, frameTuple{frameID, t.Frames[frameID]})
	}
	return frames
}

func (t Trace) findNextActiveStackID(samples []*Sample, i int) (int, int) {
	for ; i < len(samples); i++ {
		s := samples[i]
		if t.Stacks[s.StackID].IsActive() {
			return i, s.StackID
		}
	}
	return -1, -1
}

func findCommonFrames(a, b []frameTuple) []frameTuple {
	var c []frameTuple
	for i, j := len(a)-1, len(b)-1; i >= 0 && j >= 0; i, j = i-1, j-1 {
		if a[i].frame.ID() == b[j].frame.ID() {
			c = append(c, a[i])
			continue
		}
		break
	}
	reverse(c)
	return c
}

func reverse(a []frameTuple) {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}

func (s Stack) IsActive() bool {
	return len(s) != 0
}

func (t Trace) CollectFrames(stackID int) []frame.Frame {
	stack := t.Stacks[stackID]
	frames := make([]frame.Frame, 0, len(stack))
	for _, frameID := range stack {
		frames = append(frames, t.Frames[frameID])
	}
	return frames
}
