package sample

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/clientsdk"
	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/measurements"
	"github.com/getsentry/vroom/internal/metadata"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/timeutil"
	"github.com/getsentry/vroom/internal/transaction"
	"github.com/getsentry/vroom/internal/utils"
)

var (
	ErrInvalidStackID = errors.New("profile contains invalid stack id")
	ErrInvalidFrameID = errors.New("profile contains invalid frame id")
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

	Sample struct {
		ElapsedSinceStartNS uint64 `json:"elapsed_since_start_ns"`
		QueueAddress        string `json:"queue_address,omitempty"`
		StackID             int    `json:"stack_id"`
		State               State  `json:"state,omitempty"`
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

	Profile struct {
		RawProfile
	}

	RawProfile struct {
		ClientSDK           clientsdk.ClientSDK                 `json:"client_sdk"`
		DebugMeta           debugmeta.DebugMeta                 `json:"debug_meta"`
		Device              Device                              `json:"device"`
		Environment         string                              `json:"environment,omitempty"`
		EventID             string                              `json:"event_id"`
		Measurements        map[string]measurements.Measurement `json:"measurements,omitempty"`
		OS                  OS                                  `json:"os"`
		Options             utils.Options                       `json:"options,omitempty"`
		OrganizationID      uint64                              `json:"organization_id"`
		Platform            platform.Platform                   `json:"platform"`
		ProjectID           uint64                              `json:"project_id"`
		Received            timeutil.Time                       `json:"received"`
		Release             string                              `json:"release"`
		RetentionDays       int                                 `json:"retention_days"`
		Runtime             Runtime                             `json:"runtime"`
		Sampled             bool                                `json:"sampled"`
		Timestamp           time.Time                           `json:"timestamp"`
		Trace               Trace                               `json:"profile"`
		Transaction         transaction.Transaction             `json:"transaction"`
		TransactionMetadata transaction.Metadata                `json:"transaction_metadata,omitempty"`
		TransactionTags     map[string]string                   `json:"transaction_tags,omitempty"`
		Transactions        []transaction.Transaction           `json:"transactions,omitempty"`
		Version             string                              `json:"version"`
	}

	State string
)

const (
	Idle State = "idle"
)

func (p *Profile) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &p.RawProfile)
	if err != nil {
		return err
	}
	p.moveTransaction()
	return nil
}

func (p Profile) GetRelease() string {
	return p.Release
}

func (p Profile) GetDebugMeta() debugmeta.DebugMeta {
	return p.DebugMeta
}

func (q QueueMetadata) LabeledAsMainThread() bool {
	return q.Label == "com.apple.main-thread"
}

func (p Profile) GetOrganizationID() uint64 {
	return p.OrganizationID
}

func (p Profile) GetProjectID() uint64 {
	return p.ProjectID
}

func (p Profile) GetID() string {
	return p.EventID
}

func StoragePath(organizationID, projectID uint64, profileID string) string {
	return fmt.Sprintf(
		"%d/%d/%s",
		organizationID,
		projectID,
		strings.ReplaceAll(profileID, "-", ""),
	)
}

func (p Profile) StoragePath() string {
	return StoragePath(p.OrganizationID, p.ProjectID, p.EventID)
}

func (p Profile) GetPlatform() platform.Platform {
	return p.Platform
}

func (p Profile) GetEnvironment() string {
	return p.Environment
}

func (p Profile) GetTransaction() transaction.Transaction {
	return p.Transaction
}

func (p Profile) GetTimestamp() time.Time {
	return p.Timestamp
}

func (p Profile) GetReceived() time.Time {
	return p.Received.Time()
}

func (p Profile) GetRetentionDays() int {
	return p.RetentionDays
}

func (p Profile) GetDurationNS() uint64 {
	maxSampleIndex := len(p.Trace.Samples) - 1
	if maxSampleIndex < 0 {
		return 0
	}
	return p.Trace.Samples[maxSampleIndex].ElapsedSinceStartNS - p.Trace.Samples[0].ElapsedSinceStartNS
}

// CallTrees generates call trees from samples.
func (p Profile) CallTrees() (map[uint64][]*nodetree.Node, error) {
	sort.SliceStable(p.Trace.Samples, func(i, j int) bool {
		return p.Trace.Samples[i].ElapsedSinceStartNS < p.Trace.Samples[j].ElapsedSinceStartNS
	})

	activeThreadID := p.Transaction.ActiveThreadID
	treesByThreadID := make(map[uint64][]*nodetree.Node)
	samplesByThreadID := make(map[uint64][]Sample)

	for _, s := range p.Trace.Samples {
		samplesByThreadID[s.ThreadID] = append(samplesByThreadID[s.ThreadID], s)
	}

	var current *nodetree.Node
	h := fnv.New64()
	for _, samples := range samplesByThreadID {
		// The last sample is not represented, only used for its timestamp.
		for sampleIndex := 0; sampleIndex < len(samples)-1; sampleIndex++ {
			s := samples[sampleIndex]
			if s.ThreadID != activeThreadID {
				continue
			}

			if len(p.Trace.Stacks) <= s.StackID {
				return nil, ErrInvalidStackID
			}

			stack := p.Trace.Stacks[s.StackID]
			for i := len(stack) - 1; i >= 0; i-- {
				if len(p.Trace.Frames) <= stack[i] {
					return nil, ErrInvalidFrameID
				}
			}

			nextTimestamp := samples[sampleIndex+1].ElapsedSinceStartNS

			for i := len(stack) - 1; i >= 0; i-- {
				f := p.Trace.Frames[stack[i]]
				if p.Runtime.Name == "hermes" {
					f.IsReactNative = true
				}
				f.WriteToHash(h)
				fingerprint := h.Sum64()
				if current == nil {
					i := len(treesByThreadID[s.ThreadID]) - 1
					if i >= 0 && treesByThreadID[s.ThreadID][i].Fingerprint == fingerprint &&
						treesByThreadID[s.ThreadID][i].EndNS == s.ElapsedSinceStartNS {
						current = treesByThreadID[s.ThreadID][i]
						current.Update(nextTimestamp)
					} else {
						n := nodetree.NodeFromFrame(f, s.ElapsedSinceStartNS, nextTimestamp, fingerprint)
						treesByThreadID[s.ThreadID] = append(treesByThreadID[s.ThreadID], n)
						current = n
					}
				} else {
					i := len(current.Children) - 1
					if i >= 0 && current.Children[i].Fingerprint == fingerprint && current.Children[i].EndNS == s.ElapsedSinceStartNS {
						current = current.Children[i]
						current.Update(nextTimestamp)
					} else {
						n := nodetree.NodeFromFrame(f, s.ElapsedSinceStartNS, nextTimestamp, fingerprint)
						current.Children = append(current.Children, n)
						current = n
					}
				}
			}
			h.Reset()
			current = nil
		}
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
	if m, exists := t.QueueMetadata[queueAddress]; exists &&
		((m.LabeledAsMainThread() && mainThread) || !m.LabeledAsMainThread()) {
		return m.Label
	}
	return ""
}

func (p *Profile) IsSampleFormat() bool {
	return true
}

func (p *Profile) Speedscope() (speedscope.Output, error) {
	sort.SliceStable(p.Trace.Samples, func(i, j int) bool {
		return p.Trace.Samples[i].ElapsedSinceStartNS < p.Trace.Samples[j].ElapsedSinceStartNS
	})

	threadIDToProfile := make(map[uint64]*speedscope.SampledProfile)
	addressToFrameIndex := make(map[string]int)
	threadIDToPreviousTimestampNS := make(map[uint64]uint64)
	frames := make([]speedscope.Frame, 0)
	// we need to find the frame index of the main function so we can remove the frames before it
	mainFunctionFrameIndex := -1
	mainThreadID := p.Transaction.ActiveThreadID
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
					Col:  fr.Column,
					File: fr.File,
					// image exists for legacy reasons as a field coalesced from module and package
					// the speedscope transform on the sampled format is being removed, so leave
					// it alone for now
					Image:         fr.ModuleOrPackage(),
					Inline:        fr.IsInline(),
					IsApplication: fr.IsInApp(),
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
		DurationNS:         p.GetDurationNS(),
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
				DurationNS:           p.GetDurationNS(),
				Environment:          p.Environment,
				OrganizationID:       p.OrganizationID,
				Platform:             p.Platform,
				ProfileID:            p.EventID,
				ProjectID:            p.ProjectID,
				Received:             p.Received,
				Timestamp:            p.Timestamp,
				TraceID:              p.Transaction.TraceID,
				TransactionID:        p.Transaction.ID,
				TransactionName:      p.Transaction.Name,
			},
			Version: p.Release,
		},
		Platform:        p.Platform,
		ProfileID:       p.EventID,
		Profiles:        allProfiles,
		ProjectID:       p.ProjectID,
		Shared:          speedscope.SharedData{Frames: frames},
		TransactionName: p.Transaction.Name,
		Version:         p.Release,
		Measurements:    p.Measurements,
	}, nil
}

func (p *Profile) Metadata() metadata.Metadata {
	return metadata.Metadata{
		Architecture:         p.Device.Architecture,
		DeviceClassification: p.Device.Classification,
		DeviceLocale:         p.Device.Locale,
		DeviceManufacturer:   p.Device.Manufacturer,
		DeviceModel:          p.Device.Model,
		DeviceOSBuildNumber:  p.OS.BuildNumber,
		DeviceOSName:         p.OS.Name,
		DeviceOSVersion:      p.OS.Version,
		SDKName:              p.ClientSDK.Name,
		SDKVersion:           p.ClientSDK.Version,
		ID:                   p.EventID,
		ProjectID:            strconv.FormatUint(p.ProjectID, 10),
		Timestamp:            p.Timestamp.Unix(),
		TraceDurationMs:      float64(p.GetDurationNS()) / 1_000_000,
		TransactionID:        p.Transaction.ID,
		TransactionName:      p.Transaction.Name,
		VersionName:          p.Release,
	}
}

func (p *Profile) Normalize() {
	for i := range p.Trace.Frames {
		f := p.Trace.Frames[i]
		f.Normalize(p.Platform)
		p.Trace.Frames[i] = f
	}

	if p.Platform == platform.Cocoa {
		p.Trace.trimCocoaStacks()
	} else if p.Platform == platform.Python {
		p.Trace.trimPythonStacks()
	}

	p.Trace.ReplaceIdleStacks()
}

func (p *Profile) GetOptions() utils.Options {
	return p.Options
}

func (p *Profile) GetFrameWithFingerprint(target uint32) (frame.Frame, error) {
	for _, f := range p.Trace.Frames {
		if f.Fingerprint() == target {
			return f, nil
		}
	}
	return frame.Frame{}, frame.ErrFrameNotFound
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

func (t *Trace) ReplaceIdleStacks() {
	threadIDs, samplesByThreadID := t.SamplesByThreadD()

	for _, threadID := range threadIDs {
		samples := samplesByThreadID[threadID]
		previousActiveStackID := -1
		var nextActiveSampleIndex, nextActiveStackID int

		for i := 0; i < len(samples); i++ {
			s := samples[i]

			// keep track of the previous active sample as we go
			if t.Stacks[s.StackID].IsActive() {
				previousActiveStackID = s.StackID
				continue
			}

			// if there's no frame, the thread is considered idle at this time
			s.State = Idle

			// if it's an idle stack but we don't have a previous active stack
			// we keep looking
			if previousActiveStackID == -1 {
				continue
			}

			if i >= nextActiveSampleIndex {
				nextActiveSampleIndex, nextActiveStackID = t.findNextActiveStackID(samples, i)
				if nextActiveSampleIndex == -1 {
					// no more active sample on this thread
					for ; i < len(samples); i++ {
						samples[i].State = Idle
					}
					break
				}
			}

			previousFrames := t.framesList(previousActiveStackID)
			nextFrames := t.framesList(nextActiveStackID)
			commonFrames := findCommonFrames(previousFrames, nextFrames)

			// add the common stack to the list of stacks
			commonStack := make([]int, 0, len(commonFrames))
			for _, frame := range commonFrames {
				commonStack = append(commonStack, frame.index)
			}
			commonStackID := len(t.Stacks)
			t.Stacks = append(t.Stacks, commonStack)

			// replace all idle stacks until next active sample
			for ; i < nextActiveSampleIndex; i++ {
				samples[i].StackID = commonStackID
				samples[i].State = Idle
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

func (p *RawProfile) moveTransaction() {
	if len(p.Transactions) > 0 {
		p.Transaction = p.Transactions[0]
		p.Transactions = nil
	}
}

func (t *Trace) trimCocoaStacks() {
	// Find main frame index in frames
	mfi := -1
	for i, f := range t.Frames {
		if f.Function == "main" {
			mfi = i
			break
		}
	}
	// We do nothing if we don't find it
	if mfi == -1 {
		return
	}
	for si, s := range t.Stacks {
		// Find main frame index in the stack
		msi := len(s)
		// Stop searching after 10 frames, it's not there
		var until int
		if len(s) > 10 {
			until = len(s) - 10
		}
		for i := len(s) - 1; i >= until; i-- {
			fi := s[i]
			if fi == mfi {
				msi = i
				break
			}
		}
		// Skip the stack if we're already at the end or we didn't find it
		if msi >= len(s)-1 {
			continue
		}
		// Filter unsymbolicated frames after the main frame index
		ci := msi + 1
		for i := ci; i < len(s); i++ {
			fi := s[i]
			f := t.Frames[fi]
			if f.Data.SymbolicatorStatus == "symbolicated" {
				t.Stacks[si][ci] = fi
				ci++
			}
		}
		t.Stacks[si] = t.Stacks[si][:ci]
	}
}

func (t *Trace) trimPythonStacks() {
	// Find the module frame index in frames
	mfi := -1
	for i, f := range t.Frames {
		if f.File == "<string>" && f.Function == "<module>" {
			mfi = i
		}
	}

	// We do nothing if we don't find it
	if mfi == -1 {
		return
	}

	for si, s := range t.Stacks {
		l := len(s)

		// ignore empty stacks
		if l == 0 {
			continue
		}

		// found the module frame so trim it
		if s[l-1] == mfi {
			t.Stacks[si] = t.Stacks[si][:l-1]
		}
	}
}

func (p RawProfile) GetTransactionMetadata() transaction.Metadata {
	return p.TransactionMetadata
}

func (p RawProfile) GetTransactionTags() map[string]string {
	return p.TransactionTags
}

func (p RawProfile) IsSampled() bool {
	return p.Sampled
}

func (p RawProfile) GetMeasurements() map[string]measurements.Measurement {
	return p.Measurements
}

func (p *RawProfile) SetProfileID(ID string) {
	p.EventID = ID
}
