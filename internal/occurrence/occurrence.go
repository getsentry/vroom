package occurrence

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/google/uuid"
)

type (
	EvidenceName string
	IssueTitle   string

	Type int

	Evidence struct {
		Name      EvidenceName `json:"name"`
		Value     string       `json:"value"`
		Important bool         `json:"important"`
	}

	// Event holds the metadata related to a profile.
	Event struct {
		Environment    string            `json:"environment"`
		ID             string            `json:"event_id"`
		OrganizationID uint64            `json:"-"`
		Platform       platform.Platform `json:"platform"`
		ProjectID      uint64            `json:"project_id"`
		Received       time.Time         `json:"received"`
		Release        string            `json:"release,omitempty"`
		StackTrace     StackTrace        `json:"stacktrace"`
		Tags           map[string]string `json:"tags"`
		Timestamp      time.Time         `json:"timestamp"`
		Transaction    string            `json:"transaction,omitempty"`
	}

	// Occurrence represents a potential issue detected.
	Occurrence struct {
		Category        Category               `json:"-"`
		DetectionTime   time.Time              `json:"detection_time"`
		Event           Event                  `json:"event"`
		EvidenceData    map[string]interface{} `json:"evidence_data,omitempty"`
		EvidenceDisplay []Evidence             `json:"evidence_display,omitempty"`
		Fingerprint     string                 `json:"fingerprint"`
		ID              string                 `json:"id"`
		IssueTitle      IssueTitle             `json:"issue_title"`
		Level           string                 `json:"level,omitempty"`
		ResourceID      string                 `json:"resource_id,omitempty"`
		Subtitle        string                 `json:"subtitle"`
		Type            Type                   `json:"type"`

		// Only use for stats.
		durationNS  uint64
		sampleCount int
	}

	StackTrace struct {
		Frames []frame.Frame `json:"frames"`
	}
)

const (
	ProfileBlockedThreadType Type = 2000

	EvidenceNamePackage  EvidenceName = "Package"
	EvidenceNameFunction EvidenceName = "Suspect function"
)

var (
	IssueTitles = map[Category]IssueTitle{
		Compression:      "Compression on Main Thread",
		CoreDataBlock:    "Object Context operation on Main Thread",
		CoreDataMerge:    "Object Context operation on Main Thread",
		CoreDataRead:     "Object Context operation on Main Thread",
		CoreDataWrite:    "Object Context operation on Main Thread",
		FileRead:         "File I/O on Main Thread",
		FileWrite:        "File I/O on Main Thread",
		HTTP:             "Network I/O on Main Thread",
		ImageDecode:      "Image decoding on Main Thread",
		ImageEncode:      "Image decoding on Main Thread",
		JSONDecode:       "JSON decoding on Main Thread",
		JSONEncode:       "JSON encoding on Main Thread",
		MLModelInference: "Machine Learning inference on Main Thread",
		MLModelLoad:      "Machine Learning model load on Main Thread",
		Regex:            "Regex on Main Thread",
		SQL:              "SQL operation on Main Thread",
		ViewInflation:    "SwiftUI View inflation on Main Thread",
		ViewLayout:       "SwiftUI View layout on Main Thread",
		ViewRender:       "SwiftUI View render on Main Thread",
		ViewUpdate:       "SwiftUI View update on Main Thread",
		XPC:              "XPC operation on Main Thread",
	}
)

// NewOccurrence returns an Occurrence struct populated with info.
func NewOccurrence(p profile.Profile, ni nodeInfo) *Occurrence {
	t := p.Transaction()
	title, exists := IssueTitles[ni.Category]
	if !exists {
		title = "Issue detected"
	}
	h := md5.New()
	_, _ = io.WriteString(h, strconv.FormatUint(p.ProjectID(), 10))
	_, _ = io.WriteString(h, string(title))
	_, _ = io.WriteString(h, t.Name)
	_, _ = io.WriteString(h, strconv.Itoa(int(ProfileBlockedThreadType)))
	_, _ = io.WriteString(h, ni.Node.Package)
	_, _ = io.WriteString(h, ni.Node.Name)
	fingerprint := fmt.Sprintf("%x", h.Sum(nil))
	tags := buildOccurrenceTags(p)
	return &Occurrence{
		Category:      ni.Category,
		DetectionTime: time.Now().UTC(),
		Event: Event{
			Environment:    p.Environment(),
			ID:             p.ID(),
			OrganizationID: p.OrganizationID(),
			Platform:       p.Platform(),
			ProjectID:      p.ProjectID(),
			Received:       p.Received(),
			Release:        p.Release(),
			StackTrace:     StackTrace{Frames: ni.StackTrace},
			Tags:           tags,
			Timestamp:      p.Timestamp(),
			Transaction:    t.ID,
		},
		EvidenceData: map[string]interface{}{
			"frame_name":    ni.Node.Name,
			"frame_package": ni.Node.Package,
		},
		EvidenceDisplay: []Evidence{
			{
				Name:      EvidenceNameFunction,
				Value:     ni.Node.Name,
				Important: true,
			},
			{
				Name:  EvidenceNamePackage,
				Value: ni.Node.Package,
			},
		},
		Fingerprint: fingerprint,
		ID:          uuid.New().String(),
		IssueTitle:  title,
		Subtitle:    t.Name,
		Type:        ProfileBlockedThreadType,
		durationNS:  ni.Node.DurationNS,
		sampleCount: ni.Node.SampleCount,
	}
}

func buildOccurrenceTags(p profile.Profile) map[string]string {
	pm := p.Metadata()
	tags := map[string]string{
		"device_classification": pm.DeviceClassification,
		"device_locale":         pm.DeviceLocale,
		"device_manufacturer":   pm.DeviceManufacturer,
		"device_model":          pm.DeviceModel,
		"device_os_name":        pm.DeviceOSName,
		"device_os_version":     pm.DeviceOSVersion,
	}

	if pm.DeviceOSBuildNumber != "" {
		tags["device_os_build_number"] = pm.DeviceOSBuildNumber
	}

	return tags
}

func (o *Occurrence) Link() (string, error) {
	link, err := url.Parse(fmt.Sprintf("https://sentry.io/api/0/profiling/projects/%d/profile/%s/", o.Event.ProjectID, o.Event.ID))
	if err != nil {
		return "", err
	}
	params := make(url.Values)
	params.Add("package", o.EvidenceDisplay[1].Value)
	params.Add("name", o.EvidenceDisplay[0].Value)
	link.RawQuery = params.Encode()
	return link.String(), nil
}

func (o *Occurrence) Save() (map[string]bigquery.Value, string, error) {
	link, err := o.Link()
	if err != nil {
		return nil, "", err
	}
	return map[string]bigquery.Value{
		"category":        o.Category,
		"detected_at":     o.DetectionTime,
		"duration_ns":     int(o.durationNS),
		"link":            link,
		"organization_id": strconv.FormatUint(o.Event.OrganizationID, 10),
		"platform":        o.Event.Platform,
		"profile_id":      o.Event.ID,
		"project_id":      strconv.FormatUint(o.Event.ProjectID, 10),
		"sample_count":    o.sampleCount,
	}, bigquery.NoDedupeID, nil
}
