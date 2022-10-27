package measurements

type Measurement struct {
	Unit   string             `json:"unit"`
	Values []MeasurementValue `json:"values"`
}

type MeasurementValue struct {
	ElapsedSinceStartNs uint64  `json:"elapsed_since_start_ns"`
	Value               float64 `json:"value"`
}
