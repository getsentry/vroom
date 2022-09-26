package profile

import (
	"sort"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/speedscope"
)

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

func (p Python) CallTrees() map[uint64][]*nodetree.Node {
	return make(map[uint64][]*nodetree.Node)
}

func (p *Python) Speedscope() (speedscope.Output, error) {
	threadIDToProfile := make(map[uint64]*speedscope.SampledProfile)
	threadIDToPreviousTimestampNS := make(map[uint64]uint64)

	sort.Slice(p.Samples, func(i, j int) bool {
		return p.Samples[i].RelativeTimestampNS <= p.Samples[j].RelativeTimestampNS
	})
	for _, sample := range p.Samples {
		sampProfile, ok := threadIDToProfile[sample.ThreadID]
		if !ok {
			sampProfile = &speedscope.SampledProfile{
				StartValue:   sample.RelativeTimestampNS,
				ThreadID:     sample.ThreadID,
				IsMainThread: false,
				Type:         speedscope.ProfileTypeSampled,
				Unit:         speedscope.ValueUnitNanoseconds,
			}
			threadIDToProfile[sample.ThreadID] = sampProfile
		} else {
			sampProfile.Weights = append(sampProfile.Weights, sample.RelativeTimestampNS-threadIDToPreviousTimestampNS[sample.ThreadID])
		}

		samp := make([]int, 0, len(sample.Frames))
		for i := len(sample.Frames) - 1; i >= 0; i-- {
			samp = append(samp, sample.Frames[i])
		}

		sampProfile.Samples = append(sampProfile.Samples, samp)
		sampProfile.EndValue = sample.RelativeTimestampNS
		threadIDToPreviousTimestampNS[sample.ThreadID] = sample.RelativeTimestampNS
	}

	frames := make([]speedscope.Frame, 0, len(p.Frames))
	for _, pythonFrame := range p.Frames {
		frames = append(frames, speedscope.Frame{
			File: pythonFrame.File,
			Name: pythonFrame.Name,
			Line: pythonFrame.Line,
		})
	}

	var mainThreadProfileIndex int
	var mainThreadID uint64

	allProfiles := make([]interface{}, 0)
	for threadID, prof := range threadIDToProfile {
		// There is no thread metadata being sent by the python profiler at the moment,
		// so we use this heuristic to find a main thread. not perfect but good enough
		// until we start sending the metadata needed.
		if threadID < mainThreadID {
			mainThreadID = threadID
		}
		prof.Weights = append(prof.Weights, 0)
		allProfiles = append(allProfiles, prof)
	}

	return speedscope.Output{
		ActiveProfileIndex: mainThreadProfileIndex,
		Profiles:           allProfiles,
		Shared:             speedscope.SharedData{Frames: frames},
	}, nil
}
