package profile

type PythonSample struct {
	Frames              []int  `json:"frames"`
	RelativeTimestampNS uint64 `json:"relative_timestamp_ns"`
	ThreadID            uint64 `json:"thread_id"`
}

type PythonFrame struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line uint32 `json:"line"`
}

type Python struct {
	Samples []PythonSample `json:"samples"`
	Frames  []PythonFrame  `json:"frames"`
}
