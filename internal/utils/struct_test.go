package utils

import (
	"testing"

	"github.com/getsentry/vroom/internal/testutil"
)

func TestMergeContinuousProfileCandidate(t *testing.T) {
	t1 := "1"
	t2 := "2"
	tests := []struct {
		name       string
		candidates []ContinuousProfileCandidate
		want       []ContinuousProfileCandidate
	}{
		{
			name: "merge candidates",
			candidates: []ContinuousProfileCandidate{
				{
					ProjectID:  1,
					ProfilerID: "1111",
					ChunkID:    "1111",
					ThreadID:   &t1,
					Start:      100,
					End:        200,
				},
				{
					ProjectID:  1,
					ProfilerID: "2222",
					ChunkID:    "2222",
					ThreadID:   &t2,
					Start:      100,
					End:        200,
				},
				{
					ProjectID:  1,
					ProfilerID: "1111",
					ChunkID:    "1111",
					ThreadID:   &t1,
					Start:      200,
					End:        400,
				},
				{ // same chunkID but different threadID
					ProjectID:  1,
					ProfilerID: "1111",
					ChunkID:    "1111",
					ThreadID:   &t2,
					Start:      100,
					End:        300,
				},
			}, //end candidates
			want: []ContinuousProfileCandidate{
				{
					ProjectID:  1,
					ProfilerID: "1111",
					ChunkID:    "1111",
					Intervals: map[string][]Interval{
						t1: {
							{Start: 100, End: 200},
							{Start: 200, End: 400},
						},
						t2: {
							{Start: 100, End: 300},
						},
					},
				},
				{
					ProjectID:  1,
					ProfilerID: "2222",
					ChunkID:    "2222",
					Intervals: map[string][]Interval{t2: {
						{Start: 100, End: 200},
					}},
				},
			}, // end want
		},
	} // end tests

	for _, test := range tests {
		newCandidates := MergeContinuousProfileCandidate(test.candidates)
		t.Logf("%+v\n", newCandidates)
		if diff := testutil.Diff(newCandidates, test.want); diff != "" {
			t.Fatalf("Result mismatch: got - want +\n%s", diff)
		}
	}
}
