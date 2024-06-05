package chunk

import "sort"

func MergeChunks(chunks []Chunk) Chunk {
	if len(chunks) == 0 {
		return Chunk{}
	}
	sort.Slice(chunks, func(i, j int) bool {
		_, endFirstChunk := chunks[i].StartEndTimestamps()
		startSecondChunk, _ := chunks[j].StartEndTimestamps()
		return endFirstChunk <= startSecondChunk
	})

	chunk := chunks[0]
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
	}
	return chunk
}
