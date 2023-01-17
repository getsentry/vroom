package occurrence

import (
	"crypto/md5"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
)

type (
	EvidenceName   string
	OccurrenceType int

	Evidence struct {
		Name      EvidenceName `json:"name"`
		Value     string       `json:"value"`
		Important bool         `json:"important"`
	}

	// Event holds the metadata related to a profile
	Event struct {
		Environment string            `json:"environment"`
		ID          string            `json:"event_id"`
		Platform    platform.Platform `json:"platform"`
		ProjectID   uint64            `json:"project_id"`
		Received    time.Time         `json:"received"`
		Tags        map[string]string `json:"tags"`
		Timestamp   time.Time         `json:"timestamp"`
		Transaction string            `json:"transaction,omitempty"`
	}

	// Occurrence represents a potential issue detected
	Occurrence struct {
		DetectionTime   time.Time              `json:"detection_time"`
		Event           Event                  `json:"event"`
		EvidenceData    map[string]interface{} `json:"evidence_data,omitempty"`
		EvidenceDisplay []Evidence             `json:"evidence_display,omitempty"`
		Fingerprint     string                 `json:"fingerprint"`
		ID              string                 `json:"id"`
		IssueTitle      string                 `json:"issue_title"`
		Level           string                 `json:"level,omitempty"`
		ResourceID      string                 `json:"resource_id,omitempty"`
		Stacktrace      Stacktrace             `json:"stacktrace"`
		Subtitle        string                 `json:"subtitle"`
		Type            OccurrenceType         `json:"type"`
	}

	Stacktrace struct {
		Frames []frame.Frame `json:"frames"`
	}
)

const (
	ProfileBlockedThreadType OccurrenceType = 2000

	EvidenceNamePackage  EvidenceName = "Package"
	EvidenceNameFunction EvidenceName = "Suspect function"
)

func (o *Occurrence) GenerateFingerprint() error {
	h := md5.New()
	_, _ = io.WriteString(h, strconv.FormatUint(o.Event.ProjectID, 10))
	_, _ = io.WriteString(h, o.IssueTitle)
	_, _ = io.WriteString(h, o.Subtitle)
	_, _ = io.WriteString(h, strconv.Itoa(int(o.Type)))
	for _, e := range o.EvidenceDisplay {
		if e.Name == EvidenceNamePackage || e.Name == EvidenceNameFunction {
			_, _ = io.WriteString(h, e.Value)
		}
	}
	o.Fingerprint = fmt.Sprintf("%x", h.Sum(nil))
	return nil
}
