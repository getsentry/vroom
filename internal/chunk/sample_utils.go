package chunk

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/getsentry/vroom/internal/measurements"
	"gocloud.dev/blob"
)

func MergeSampleChunks(chunks []SampleChunk, startTS, endTS uint64) (SampleChunk, error) {
	if len(chunks) == 0 {
		return SampleChunk{}, nil
	}
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].EndTimestamp() <= chunks[j].StartTimestamp()
	})

	mergedMeasurement := make(map[string]measurements.MeasurementV2)

	start := float64(startTS) / 1e9
	end := float64(endTS) / 1e9

	chunk := chunks[0]
	if len(chunk.Measurements) > 0 {
		err := json.Unmarshal(chunk.Measurements, &mergedMeasurement)
		if err != nil {
			return SampleChunk{}, err
		}
	}

	// clean up the samples in the first chunk
	samples := make([]Sample, 0, len(chunk.Profile.Samples))
	for _, sample := range chunk.Profile.Samples {
		if sample.Timestamp < start || sample.Timestamp > end {
			// sample from chunk lies outside start/end range so skip it
			continue
		}
		samples = append(samples, sample)
	}

	for i := 1; i < len(chunks); i++ {
		c := chunks[i]
		// Update all the frame indices of the chunk we're going to add/merge
		// to the first one.
		// If the first chunk had a couple of frames, and the second chunk too,
		// then all the stacks in the second chunk that refers to frames at index
		// fr[0] and fr[1], once merged should refer to frames at index fr[2], fr[3].
		for j, stack := range c.Profile.Stacks {
			for z, frameID := range stack {
				c.Profile.Stacks[j][z] = frameID + len(chunk.Profile.Frames)
			}
		}
		chunk.Profile.Frames = append(chunk.Profile.Frames, c.Profile.Frames...)
		// The same goes for chunk samples stack IDs
		for j, sample := range c.Profile.Samples {
			c.Profile.Samples[j].StackID = sample.StackID + len(chunk.Profile.Stacks)
		}
		chunk.Profile.Stacks = append(chunk.Profile.Stacks, c.Profile.Stacks...)
		for _, sample := range c.Profile.Samples {
			if sample.Timestamp < start || sample.Timestamp > end {
				// sample from chunk lies outside start/end range so skip it
				continue
			}
			samples = append(samples, sample)
		}

		// Update threadMetadata
		for k, threadMetadata := range c.Profile.ThreadMetadata {
			if _, ok := chunk.Profile.ThreadMetadata[k]; !ok {
				chunk.Profile.ThreadMetadata[k] = threadMetadata
			}
		}

		// In case we have measurements, merge them too
		if len(c.Measurements) > 0 {
			var chunkMeasurements map[string]measurements.MeasurementV2
			err := json.Unmarshal(c.Measurements, &chunkMeasurements)
			if err != nil {
				return SampleChunk{}, err
			}
			for k, measurement := range chunkMeasurements {
				if el, ok := mergedMeasurement[k]; ok {
					el.Values = append(el.Values, measurement.Values...)
					mergedMeasurement[k] = el
				} else {
					mergedMeasurement[k] = measurement
				}
			}
		}
	}

	chunk.Profile.Samples = samples

	if len(mergedMeasurement) > 0 {
		jsonRawMesaurement, err := json.Marshal(mergedMeasurement)
		if err != nil {
			return SampleChunk{}, err
		}
		chunk.Measurements = jsonRawMesaurement
	}

	return chunk, nil
}

// The task the workers expect as input.
//
// Result: the channel used to send back the output.
type TaskInput struct {
	Ctx            context.Context
	ProfilerID     string
	ChunkID        string
	OrganizationID uint64
	ProjectID      uint64
	Storage        *blob.Bucket
	Result         chan<- SampleTaskOutput
}

// The output sent back by the worker.
type SampleTaskOutput struct {
	Err   error
	Chunk SampleChunk
}
