package measurements

type Measurement struct {
	ElapsedSinceStartNs uint64  `json:"elapsed_since_start_ns"`
	Unit                string  `json:"unit"`
	Value               float64 `json:"value"`
}
