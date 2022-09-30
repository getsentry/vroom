package sample

import (
	"fmt"
	"hash"
	"hash/fnv"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

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
		ID              string `json:"id"`
		Name            string `json:"name"`
		RelativeEndNS   uint64 `json:"relative_end_ns,string"`
		RelativeStartNS uint64 `json:"relative_start_ns,string"`
		TraceID         string `json:"trace_id"`
	}

	Sample struct {
		ElapsedSinceStartNS uint64 `json:"elapsed_since_start_ns,string"`
		StackID             int    `json:"stack_id"`
		ThreadID            uint64 `json:"thread_id,string"`
	}

	Frame struct {
		File            string `json:"filename,omitempty"`
		Function        string `json:"function,omitempty"`
		InstructionAddr string `json:"instruction_addr,omitempty"`
		Lang            string `json:"lang,omitempty"`
		Line            uint32 `json:"lineno,omitempty"`
		Package         string `json:"package,omitempty"`
		Path            string `json:"abs_path,omitempty"`
		Status          string `json:"status,omitempty"`
		SymAddr         string `json:"sym_addr,omitempty"`
		Symbol          string `json:"symbol,omitempty"`
	}

	Trace struct {
		Frames  []Frame  `json:"frames"`
		Samples []Sample `json:"samples"`
		Stacks  [][]int  `json:"stacks"`
	}

	SampleProfile struct {
		DebugMeta      interface{}   `json:"debug_meta,omitempty"`
		Device         Device        `json:"device"`
		Environment    string        `json:"environment,omitempty"`
		EventID        string        `json:"event_id"`
		OS             OS            `json:"os"`
		OrganizationID uint64        `json:"organization_id"`
		Platform       string        `json:"platform"`
		ProjectID      uint64        `json:"project_id"`
		Received       time.Time     `json:"received"`
		Release        string        `json:"release"`
		Runtime        Runtime       `json:"runtime"`
		Timestamp      time.Time     `json:"timestamp"`
		Trace          Trace         `json:"profile"`
		Transactions   []Transaction `json:"transactions"`
		Version        string        `json:"version"`
	}
)

func (f Frame) PackageBaseName() string {
	if f.Package == "" {
		return ""
	}
	return path.Base(f.Package)
}

func (f Frame) WriteToHash(h hash.Hash) {
	if f.Package == "" {
		h.Write([]byte("-"))
	} else {
		h.Write([]byte(f.PackageBaseName()))
	}
	if f.Function == "" {
		h.Write([]byte("-"))
	} else {
		h.Write([]byte(f.Function))
	}
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
	sort.Slice(p.Trace.Samples, func(i, j int) bool {
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
	fmt.Println(trees)
	return trees, nil
}

func (p *SampleProfile) Speedscope() (speedscope.Output, error) {
	return speedscope.Output{}, nil
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

func (p *SampleProfile) IsApplicationPackage(pkg string) bool {
	switch p.Platform {
	case "cocoa":
		return packageutil.IsIOSApplicationPackage(pkg)
	case "rust":
		return packageutil.IsRustApplicationPackage(pkg)
	}
	return true
}
