package chrometrace

type (
	frame struct {
		Col           int    `json:"col,omitempty"`
		File          string `json:"file,omitempty"`
		Image         string `json:"image,omitempty"`
		IsApplication bool   `json:"is_application"`
		Line          int    `json:"line,omitempty"`
		Name          string `json:"name"`
	}

	event struct {
		Type  eventType `json:"type"`
		Frame int       `json:"frame"`
		At    uint64    `json:"at"`
	}

	queue struct {
		Label   string `json:"name"`
		StartNS uint64 `json:"start_ns"`
		EndNS   uint64 `json:"end_ns"`
	}

	eventedProfile struct {
		EndValue   uint64      `json:"endValue"`
		Events     []event     `json:"events"`
		Name       string      `json:"name"`
		StartValue uint64      `json:"startValue"`
		ThreadID   uint64      `json:"threadID"`
		Type       profileType `json:"type"`
		Unit       valueUnit   `json:"unit"`
	}

	sampledProfile struct {
		EndValue     uint64           `json:"endValue"`
		IsMainThread bool             `json:"isMainThread"`
		Name         string           `json:"name"`
		Priority     int              `json:"priority"`
		Queues       map[string]queue `json:"queues,omitempty"`
		Samples      [][]int          `json:"samples"`
		StartValue   uint64           `json:"startValue"`
		ThreadID     uint64           `json:"threadID"`
		Type         profileType      `json:"type"`
		Unit         valueUnit        `json:"unit"`
		Weights      []uint64         `json:"weights"`
	}

	sharedData struct {
		Frames []frame `json:"frames"`
	}

	valueUnit   string
	eventType   string
	profileType string
)

const (
	valueUnitNanoseconds valueUnit = "nanoseconds"

	eventTypeOpenFrame  eventType = "O"
	eventTypeCloseFrame eventType = "C"

	profileTypeEvented profileType = "evented"
	profileTypeSampled profileType = "sampled"
)
