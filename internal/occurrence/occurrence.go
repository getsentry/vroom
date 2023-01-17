package occurrence

import (
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
)

type (
	OccurrenceType int

	Evidence struct {
		Name      string `json:"name"`
		Value     string `json:"value"`
		Important bool   `json:"important"`
	}

	// Event holds the metadata related to a profile
	Event struct {
		ID        string            `json:"event_id"`
		Platform  platform.Platform `json:"platform"`
		ProjectID uint64            `json:"project_id"`
		Received  time.Time         `json:"received"`
		Tags      map[string]string `json:"tags,omitempty"`
		Timestamp time.Time         `json:"timestamp"`
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
		Subtitle        string                 `json:"subtitle"`
		Type            OccurrenceType         `json:"type"`
		Stacktrace      Stacktrace             `json:"stacktrace"`
	}

	Stacktrace struct {
		Frames []frame.Frame `json:"frames"`
	}
)

const (
	ProfileBlockedThreadType OccurrenceType = 2000
)

func (o *Occurrence) GenerateFingerprint() error {
	var s strings.Builder
	s.WriteString(strconv.FormatUint(o.Event.ProjectID, 10))
	s.WriteString(o.IssueTitle)
	s.WriteString(o.Subtitle)
	s.WriteString(strconv.Itoa(int(o.Type)))
	o.Fingerprint = s.String()
	return nil
}
