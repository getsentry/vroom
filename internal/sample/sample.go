package sample

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/nodetree"
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
		RelativeEndNS   uint64 `json:"relative_end_ns"`
		RelativeStartNS uint64 `json:"relative_start_ns"`
		TraceID         string `json:"trace_id"`
	}

	Sample struct {
		RelativeTimestampNS int `json:"relative_timestamp_ns"`
		StackID             int `json:"stack_id"`
	}

	Frame struct {
		Function        string `json:"function"`
		InstructionAddr string `json:"instruction_addr"`
		Line            int    `json:"line"`
	}

	Trace struct {
		Frames  []Frame  `json:"frames"`
		Samples []Sample `json:"samples"`
		Stacks  [][]int  `json:"stacks"`
	}

	SampleProfile struct {
		sampleProfile
	}

	sampleProfile struct {
		DebugMeta      interface{} `json:"debug_meta,omitempty"`
		Device         Device      `json:"device"`
		Environment    string      `json:"environment,omitempty"`
		EventID        string      `json:"event_id"`
		OS             OS          `json:"os"`
		OrganizationID uint64      `json:"organization_id"`
		Platform       string      `json:"platform"`
		ProjectID      uint64      `json:"project_id"`
		Received       time.Time   `json:"received"`
		Release        string      `json:"release"`
		Runtime        Runtime     `json:"runtime"`
		Timestamp      time.Time   `json:"timestamp"`
		Trace          Trace       `json:"profile"`
	}
)

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
	return make(map[uint64][]*nodetree.Node), nil
}

func (p *SampleProfile) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &p.sampleProfile)
}

func (p *SampleProfile) Speedscope() (speedscope.Output, error) {
	return speedscope.Output{}, nil
}
