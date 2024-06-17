package measurements

type Measurement struct {
	Unit   string             `json:"unit"`
	Values []MeasurementValue `json:"values"`
}

type MeasurementValue struct {
	ElapsedSinceStartNs uint64  `json:"elapsed_since_start_ns"`
	Value               float64 `json:"value"`
}

type MeasurementV2 struct {
	Unit   string               `json:"unit"`
	Values []MeasurementValueV2 `json:"values"`
}

// https://github.com/getsentry/relay/blob/master/relay-profiling/src/measurements.rs#L23-L29
type MeasurementValueV2 struct {
	// UNIX timestamp in seconds as a float
	Timestamp float64 `json:"timestamp"`
	Value     float64 `json:"value"`
}
