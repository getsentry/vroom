package speedscope

import (
	"testing"
)

func TestSortSamplesAlphabetically(t *testing.T) {
	frames := []Frame{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
		{Name: "d"},
	}

	samples := [][]int{
		{0, 3},
		{1, 3},
		{0, 1, 2},
		{0, 1, 2, 3},
	}

	sortedSamples := [][]int{
		{0, 1, 2},
		{0, 1, 2, 3},
		{0, 3},
		{1, 3},
	}

	SortSamplesAlphabetically(samples, frames)

	for i := 0; i < len(samples); i++ {
		if len(samples[i]) != len(sortedSamples[i]) {
			t.Fatalf("the 2 stacks have different size: len(samples[%d])=%d and len(samples[%d])=%d",
				i, len(sortedSamples[i]), i, len(sortedSamples[i]))
		} else {
			for j := 0; j < len(samples[i]); j++ {
				if samples[i][j] != sortedSamples[i][j] {
					t.Fatalf("stack sample %d differ for samples and sortedSamples", i)
				}
			}
		}
	}
}
