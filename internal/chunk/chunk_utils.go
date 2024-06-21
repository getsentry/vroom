package chunk

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/getsentry/vroom/internal/measurements"
	"gocloud.dev/blob"
)

func MergeChunks(chunks []Chunk) (Chunk, error) {
	if len(chunks) == 0 {
		return Chunk{}, nil
	}
	sort.Slice(chunks, func(i, j int) bool {
		_, endFirstChunk := chunks[i].StartEndTimestamps()
		startSecondChunk, _ := chunks[j].StartEndTimestamps()
		return endFirstChunk <= startSecondChunk
	})

	var mergedMeasurement = make(map[string]measurements.MeasurementV2)

	chunk := chunks[0]
	if len(chunk.Measurements) > 0 {
		err := json.Unmarshal(chunk.Measurements, &mergedMeasurement)
		if err != nil {
			return Chunk{}, err
		}
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
		chunk.Profile.Samples = append(chunk.Profile.Samples, c.Profile.Samples...)

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
				return Chunk{}, err
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

	if len(mergedMeasurement) > 0 {
		jsonRawMesaurement, err := json.Marshal(mergedMeasurement)
		if err != nil {
			return Chunk{}, err
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
	Result         chan<- TaskOutput
}

// The output sent back by the worker.
type TaskOutput struct {
	Err   error
	Chunk Chunk
}
