package measurements

import "github.com/getsentry/vroom/internal/types"

type (
	Measurement struct {
		Unit   string             `json:"unit"`
		Values []MeasurementValue `json:"values"`
	}

	MeasurementValue struct {
		ElapsedSinceStartNs types.Uint64 `json:"elapsed_since_start_ns"`
		Value               float64      `json:"value"`
	}
)
