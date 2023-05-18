package occurrence

import (
	"crypto/md5"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/getsentry/vroom/internal/android"
	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
)

type (
	EvidenceName string
	IssueTitle   string
	Type         int
	Context      string

	Evidence struct {
		Name      EvidenceName `json:"name"`
		Value     string       `json:"value"`
		Important bool         `json:"important"`
	}

	// Event holds the metadata related to a profile.
	Event struct {
		Contexts       map[Context]interface{} `json:"contexts,omitempty"`
		DebugMeta      debugmeta.DebugMeta     `json:"debug_meta"`
		Environment    string                  `json:"environment,omitempty"`
		ID             string                  `json:"event_id"`
		OrganizationID uint64                  `json:"-"`
		Platform       platform.Platform       `json:"platform"`
		ProjectID      uint64                  `json:"project_id"`
		Received       time.Time               `json:"received"`
		Release        string                  `json:"release,omitempty"`
		StackTrace     StackTrace              `json:"stacktrace"`
		Tags           map[string]string       `json:"tags"`
		Timestamp      time.Time               `json:"timestamp"`
	}

	// Occurrence represents a potential issue detected.
	Occurrence struct {
		Culprit         string                 `json:"culprit"`
		DetectionTime   time.Time              `json:"detection_time"`
		Event           Event                  `json:"event"`
		EvidenceData    map[string]interface{} `json:"evidence_data,omitempty"`
		EvidenceDisplay []Evidence             `json:"evidence_display,omitempty"`
		Fingerprint     string                 `json:"fingerprint"`
		ID              string                 `json:"id"`
		IssueTitle      IssueTitle             `json:"issue_title"`
		Level           string                 `json:"level,omitempty"`
		ProjectID       uint64                 `json:"project_id"`
		ResourceID      string                 `json:"resource_id,omitempty"`
		Subtitle        string                 `json:"subtitle"`
		Type            Type                   `json:"type"`

		// Only use for stats.
		category    Category
		durationNS  uint64
		sampleCount int
	}

	StackTrace struct {
		Frames []frame.Frame `json:"frames"`
	}

	CategoryMetadata struct {
		IssueTitle IssueTitle
		Type       Type
	}
)

const (
	NoneType        Type = 0
	FileIOType      Type = 2001
	ImageDecodeType Type = 2002
	JSONDecodeType  Type = 2003

	EvidenceNameDuration EvidenceName = "Duration"
	EvidenceNameFunction EvidenceName = "Suspect function"
	EvidenceNamePackage  EvidenceName = "Package"

	ContextTrace Context = "trace"

	ProfileID string = "profile_id"
)

var issueTitles = map[Category]CategoryMetadata{
	Base64Decode:     {IssueTitle: "Base64 Decode on Main Thread"},
	Base64Encode:     {IssueTitle: "Base64 Encode on Main Thread"},
	Compression:      {IssueTitle: "Compression on Main Thread"},
	CoreDataBlock:    {IssueTitle: "Object Context operation on Main Thread"},
	CoreDataMerge:    {IssueTitle: "Object Context operation on Main Thread"},
	CoreDataRead:     {IssueTitle: "Object Context operation on Main Thread"},
	CoreDataWrite:    {IssueTitle: "Object Context operation on Main Thread"},
	Decompression:    {IssueTitle: "Decompression on Main Thread"},
	FileRead:         {IssueTitle: "File I/O on Main Thread"},
	FileWrite:        {IssueTitle: "File I/O on Main Thread"},
	HTTP:             {IssueTitle: "Network I/O on Main Thread"},
	ImageDecode:      {IssueTitle: "Image Decoding on Main Thread", Type: ImageDecodeType},
	ImageEncode:      {IssueTitle: "Image Encoding on Main Thread"},
	JSONDecode:       {IssueTitle: "JSON Decoding on Main Thread", Type: JSONDecodeType},
	JSONEncode:       {IssueTitle: "JSON Encoding on Main Thread"},
	MLModelInference: {IssueTitle: "Machine Learning inference on Main Thread"},
	MLModelLoad:      {IssueTitle: "Machine Learning model load on Main Thread"},
	Regex:            {IssueTitle: "Regex on Main Thread"},
	SQL:              {IssueTitle: "SQL operation on Main Thread"},
	SourceContext:    {IssueTitle: "Adding Source Context is slow"},
	ThreadWait:       {IssueTitle: "Thread Wait on Main Thread"},
	ViewInflation:    {IssueTitle: "SwiftUI View inflation on Main Thread"},
	ViewLayout:       {IssueTitle: "SwiftUI View layout on Main Thread"},
	ViewRender:       {IssueTitle: "SwiftUI View render on Main Thread"},
	ViewUpdate:       {IssueTitle: "SwiftUI View update on Main Thread"},
	XPC:              {IssueTitle: "XPC operation on Main Thread"},
}

// NewOccurrence returns an Occurrence struct populated with info.
func NewOccurrence(p profile.Profile, ni nodeInfo) *Occurrence {
	t := p.Transaction()
	var title IssueTitle
	var issueType Type
	cm, exists := issueTitles[ni.Category]
	if exists {
		issueType = cm.Type
		title = cm.IssueTitle
	} else {
		issueType = NoneType
		title = IssueTitle(fmt.Sprintf("%v issue detected", ni.Category))
	}
	pf := p.Platform()
	nodeDuration := time.Duration(ni.Node.DurationNS).Round(10 * time.Microsecond)
	profilePercentage := float64(ni.Node.DurationNS*100) / float64(p.DurationNS())
	evidenceData := map[string]interface{}{
		"frame_duration_ns":   ni.Node.DurationNS,
		"frame_module":        ni.Node.Frame.Module,
		"frame_name":          ni.Node.Name,
		"frame_package":       ni.Node.Frame.Package,
		"profile_duration_ns": p.DurationNS(),
		"template_name":       "profile",
		"transaction_id":      t.ID,
		"transaction_name":    t.Name,
		ProfileID:             p.ID(),
	}
	var duration string
	switch pf {
	case platform.Android:
		duration = fmt.Sprintf(
			"%s (%0.2f%% of the profile)",
			nodeDuration,
			profilePercentage,
		)
		pf = platform.Java
		normalizeAndroidStackTrace(ni.StackTrace)
		ni.Node.Name = android.StripPackageNameFromFullMethodName(
			ni.Node.Name,
			ni.Node.Package,
		)
	default:
		duration = fmt.Sprintf(
			"%s (%0.2f%% of the profile, found in %d samples)",
			nodeDuration,
			profilePercentage,
			ni.Node.SampleCount,
		)
		evidenceData["sample_count"] = ni.Node.SampleCount
	}
	h := md5.New()
	_, _ = io.WriteString(h, strconv.FormatUint(p.ProjectID(), 10))
	_, _ = io.WriteString(h, string(title))
	_, _ = io.WriteString(h, strconv.Itoa(int(issueType)))
	_, _ = io.WriteString(h, ni.Node.Frame.ModuleOrPackage())
	_, _ = io.WriteString(h, ni.Node.Name)
	fingerprint := fmt.Sprintf("%x", h.Sum(nil))
	tags := p.TransactionTags()
	if tags == nil {
		tags = make(map[string]string)
	}
	return &Occurrence{
		Culprit:       t.Name,
		DetectionTime: time.Now().UTC(),
		Event: Event{
			DebugMeta:      p.DebugMeta(),
			Environment:    p.Environment(),
			ID:             eventID(),
			OrganizationID: p.OrganizationID(),
			Platform:       pf,
			ProjectID:      p.ProjectID(),
			Received:       p.Received(),
			Release:        p.Release(),
			StackTrace:     StackTrace{Frames: ni.StackTrace},
			Tags:           tags,
			Timestamp:      p.Timestamp(),
		},
		EvidenceData: evidenceData,
		EvidenceDisplay: []Evidence{
			{
				Important: true,
				Name:      EvidenceNameFunction,
				Value:     ni.Node.Name,
			},

			{
				Name:  EvidenceNamePackage,
				Value: ni.Node.Package,
			},
			{
				Name:  EvidenceNameDuration,
				Value: duration,
			},
		},
		Fingerprint: fingerprint,
		ID:          eventID(),
		IssueTitle:  title,
		Level:       "info",
		ProjectID:   p.ProjectID(),
		Subtitle:    ni.Node.Name,
		Type:        issueType,
		category:    ni.Category,
		durationNS:  ni.Node.DurationNS,
		sampleCount: ni.Node.SampleCount,
	}
}

func eventID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func normalizeAndroidStackTrace(st []frame.Frame) {
	for i := range st {
		st[i].Function = android.StripPackageNameFromFullMethodName(st[i].Function, st[i].Package)
	}
}
