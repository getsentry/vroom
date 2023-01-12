package occurrence

import (
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/platform"
)

type (
	// Event holds the metadata related to a profile
	Event struct {
		ID        string            `json:"event_id"`
		Platform  platform.Platform `json:"platform"`
		ProjectID uint64            `json:"project_id"`
		Tags      [][2]string       `json:"tags"`
		Timestamp time.Time         `json:"timestamp"`
	}

	// Occurrence represents a potential issue detected
	Occurrence struct {
		DetectionTime   time.Time `json:"detection_time"`
		Event           Event     `json:"event"`
		EvidenceData    string    `json:"evidence_data,omitempty"`
		EvidenceDisplay string    `json:"evidence_display,omitempty"`
		Fingerprint     string    `json:"fingerprint"`
		ID              string    `json:"id"`
		IssueTitle      string    `json:"issue_title"`
		ResourceID      string    `json:"resource_id,omitempty"`
		Subtitle        string    `json:"subtitle"`
		Type            string    `json:"type"`
	}
)

func (o *Occurrence) GenerateFingerprint() error {
	var s strings.Builder
	s.WriteString(o.IssueTitle)
	s.WriteString(o.Subtitle)
	s.WriteString(strconv.FormatUint(o.Event.ProjectID, 10))
	o.Fingerprint = s.String()
	return nil
}
