package sample

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/fnv"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/measurements"
	"github.com/getsentry/vroom/internal/metadata"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/packageutil"
	"github.com/getsentry/vroom/internal/speedscope"
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
		ThreadID            uint64 `json:"thread_id"`
	}

	Frame struct {
		Column          uint32 `json:"colno,omitempty"`
		File            string `json:"filename,omitempty"`
		Function        string `json:"function,omitempty"`
		InApp           bool   `json:"in_app"`
		InstructionAddr string `json:"instruction_addr,omitempty"`
		Lang            string `json:"lang,omitempty"`
		Line            uint32 `json:"lineno,omitempty"`
		Module          string `json:"module,omitempty"`
		Package         string `json:"package,omitempty"`
		Path            string `json:"abs_path,omitempty"`
		Status          string `json:"status,omitempty"`
		SymAddr         string `json:"sym_addr,omitempty"`
		Symbol          string `json:"symbol,omitempty"`
	}

	ThreadMetadata struct {
		Name     string `json:"name,omitempty"`
		Priority int    `json:"priority,omitempty"`
	}

	QueueMetadata struct {
		Label string `json:"label"`
	}

	Trace struct {
		Frames         []Frame                   `json:"frames"`
		QueueMetadata  map[string]QueueMetadata  `json:"queue_metadata"`
		Samples        []Sample                  `json:"samples"`
		Stacks         [][]int                   `json:"stacks"`
		ThreadMetadata map[string]ThreadMetadata `json:"thread_metadata"`
	}

	SampleProfile struct {
		DebugMeta      debugmeta.DebugMeta                 `json:"debug_meta"`
		Device         Device                              `json:"device"`
		Environment    string                              `json:"environment,omitempty"`
		EventID        string                              `json:"event_id"`
		OS             OS                                  `json:"os"`
		OrganizationID uint64                              `json:"organization_id"`
		Platform       string                              `json:"platform"`
		ProjectID      uint64                              `json:"project_id"`
		Received       time.Time                           `json:"received"`
		Release        string                              `json:"release"`
		Runtime        Runtime                             `json:"runtime"`
		Timestamp      time.Time                           `json:"timestamp"`
		Trace          Trace                               `json:"profile"`
		Transactions   []Transaction                       `json:"transactions"`
		Version        string                              `json:"version"`
		Measurements   map[string]measurements.Measurement `json:"measurements,omitempty"`
	}
)

func (q QueueMetadata) LabeledAsMainThread() bool {
	return q.Label == "com.apple.main-thread"
}

// IsMain returns true if the function is considered the main function.
// It also returns an offset indicate if we need to keep the previous frame or not.
func (f Frame) IsMain() (bool, int) {
	if f.Status != "symbolicated" {
		return false, 0
	} else if f.Function == "main" {
		return true, 0
	} else if f.Function == "UIApplicationMain" {
		return true, -1
	}
	return false, 0
}

func (f Frame) ID() string {
	// When we have a symbolicated frame we can't rely on symbol_address
	// to uniquely identify a frame since the following might happen:
	//
	// frame 1 has: sym_addr: 1, file: a.rs, line 2
	// frame 2 has: sym_addr: 1, file: a.rs, line: 4
	// because they have the same sym addr the second frame is reusing the first one,
	// and gets the wrong line number
	//
	// Also, when a frame is symbolicated but is missing the symbol_address
	// we know we're dealing with inlines, but we can't rely on instruction_address
	// neither as the inlines are all using the same one. If we were to return this
	// address in speedscope we would only generate a new frame for the parent one
	// and for the inlines we would show the same information of the parents instead
	// of their own
	//
	// As a solution here we use the following hash function that guarantees uniqueness
	// when all the information required is available
	hash := md5.Sum([]byte(fmt.Sprintf("%s:%s:%d:%s", f.File, f.Function, f.Line, f.InstructionAddr)))
	return hex.EncodeToString(hash[:])
}

func (f Frame) PackageBaseName() string {
	if f.Module != "" {
		return f.Module
	} else if f.Package != "" {
		return path.Base(f.Package)
	}
	return ""
}

func (f Frame) WriteToHash(h hash.Hash) {
	var s string
	if f.Package != "" {
		s = f.PackageBaseName()
	} else if f.File != "" {
		s = f.File
	} else {
		s = "-"
	}
	h.Write([]byte(s))
	if f.Function != "" {
		s = f.Function
	} else {
		s = "-"
	}
	h.Write([]byte(s))
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

func (p SampleProfile) GetPlatform() string {
	return p.Platform
}

func (p SampleProfile) CallTrees() (map[uint64][]*nodetree.Node, error) {
	sort.SliceStable(p.Trace.Samples, func(i, j int) bool {
		return p.Trace.Samples[i].ElapsedSinceStartNS < p.Trace.Samples[j].ElapsedSinceStartNS
	})

	trees := make(map[uint64][]*nodetree.Node)
	previousTimestamp := make(map[uint64]uint64)

	var current *nodetree.Node
	h := fnv.New64()
	for _, s := range p.Trace.Samples {
		stack := p.Trace.Stacks[s.StackID]
		for i := len(stack) - 1; i >= 0; i-- {
			f := p.Trace.Frames[stack[i]]
			f.WriteToHash(h)
			fingerprint := h.Sum64()
			if current == nil {
				i := len(trees[s.ThreadID]) - 1
				if i >= 0 && trees[s.ThreadID][i].Fingerprint == fingerprint && trees[s.ThreadID][i].EndNS == previousTimestamp[s.ThreadID] {
					current = trees[s.ThreadID][i]
					current.SetDuration(s.ElapsedSinceStartNS)
				} else {
					n := nodetree.NodeFromFrame(f.PackageBaseName(), f.Function, f.Path, f.Line, previousTimestamp[s.ThreadID], s.ElapsedSinceStartNS, fingerprint, p.IsApplicationPackage(f.Package))
					trees[s.ThreadID] = append(trees[s.ThreadID], n)
					current = n
				}
			} else {
				i := len(current.Children) - 1
				if i >= 0 && current.Children[i].Fingerprint == fingerprint && current.Children[i].EndNS == previousTimestamp[s.ThreadID] {
					current = current.Children[i]
					current.SetDuration(s.ElapsedSinceStartNS)
				} else {
					n := nodetree.NodeFromFrame(f.PackageBaseName(), f.Function, f.Path, f.Line, previousTimestamp[s.ThreadID], s.ElapsedSinceStartNS, fingerprint, p.IsApplicationPackage(f.Package))
					current.Children = append(current.Children, n)
					current = n
				}
			}
		}
		h.Reset()
		previousTimestamp[s.ThreadID] = s.ElapsedSinceStartNS
		current = nil
	}
	return trees, nil
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
		queueMetadata, qmExists := p.Trace.QueueMetadata[sample.QueueAddress]
		if !exists {
			threadMetadata, tmExists := p.Trace.ThreadMetadata[threadID]
			threadName := threadMetadata.Name
			if threadName == "" && qmExists && (!queueMetadata.LabeledAsMainThread() || sample.ThreadID != mainThreadID) {
				threadName = queueMetadata.Label
			}
			speedscopeProfile = &speedscope.SampledProfile{
				IsMainThread: sample.ThreadID == mainThreadID,
				Images:       p.DebugMeta.Images,
				Name:         threadName,
				Queues:       make(map[string]speedscope.Queue),
				StartValue:   sample.ElapsedSinceStartNS,
				ThreadID:     sample.ThreadID,
				Type:         speedscope.ProfileTypeSampled,
				Unit:         speedscope.ValueUnitNanoseconds,
			}
			if qmExists {
				speedscopeProfile.Queues[queueMetadata.Label] = speedscope.Queue{Label: queueMetadata.Label, StartNS: sample.ElapsedSinceStartNS, EndNS: sample.ElapsedSinceStartNS}
			}
			if tmExists {
				speedscopeProfile.Priority = threadMetadata.Priority
			}
			threadIDToProfile[sample.ThreadID] = speedscopeProfile
		} else {
			if qmExists {
				q, qExists := speedscopeProfile.Queues[queueMetadata.Label]
				if !qExists {
					speedscopeProfile.Queues[queueMetadata.Label] = speedscope.Queue{Label: queueMetadata.Label, StartNS: sample.ElapsedSinceStartNS, EndNS: sample.ElapsedSinceStartNS}
				} else {
					q.EndNS = sample.ElapsedSinceStartNS
					speedscopeProfile.Queues[queueMetadata.Label] = q
				}
			}
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
					Inline:        fr.Status == "symbolicated" && fr.SymAddr == "",
					IsApplication: fr.InApp || p.IsApplicationPackage(fr.Path),
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
	}, nil
}

func (p *SampleProfile) Metadata() metadata.Metadata {
	return metadata.Metadata{
		DeviceClassification: p.Device.Classification,
		DeviceLocale:         p.Device.Locale,
		DeviceManufacturer:   p.Device.Manufacturer,
		DeviceModel:          p.Device.Model,
		DeviceOsBuildNumber:  p.OS.BuildNumber,
		DeviceOsName:         p.OS.Name,
		DeviceOsVersion:      p.OS.Version,
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

func (p *SampleProfile) IsApplicationPackage(path string) bool {
	switch p.Platform {
	case "cocoa":
		return packageutil.IsIOSApplicationPackage(path)
	case "rust":
		return packageutil.IsRustApplicationPackage(path)
	}
	return true
}
