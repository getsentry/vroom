package chunk

import (
	"encoding/json"
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/testutil"
)

func TestMergeChunks(t *testing.T) {
	tests := []struct {
		name string
		have []Chunk
		want Chunk
	}{
		{
			name: "contiguous chunks",
			have: []Chunk{
				{
					Profile: Data{
						Frames: []frame.Frame{
							{Function: "c"},
							{Function: "d"},
						},
						Samples: []Sample{
							{StackID: 0, Timestamp: 2.0},
							{StackID: 1, Timestamp: 3.0},
						},
						Stacks: [][]int{
							{0, 1},
							{0, 1},
						},
					},
					Measurements: json.RawMessage(`{"first_metric":{"unit":"ms","values":[{"elapsed_since_start_ns":200,"value":1.2}]}}`),
				},
				// other chunk
				{
					Profile: Data{
						Frames: []frame.Frame{
							{Function: "a"},
							{Function: "b"},
						},
						Samples: []Sample{
							{StackID: 0, Timestamp: 0.0},
							{StackID: 1, Timestamp: 1.0},
						},
						Stacks: [][]int{
							{0, 1},
							{0, 1},
						},
					},
					Measurements: json.RawMessage(`{"first_metric":{"unit":"ms","values":[{"elapsed_since_start_ns":100,"value":1}]}}`),
				},
			},
			want: Chunk{
				Profile: Data{
					Frames: []frame.Frame{
						{Function: "a"},
						{Function: "b"},
						{Function: "c"},
						{Function: "d"},
					},
					Samples: []Sample{
						{StackID: 0, Timestamp: 0.0},
						{StackID: 1, Timestamp: 1.0},
						{StackID: 2, Timestamp: 2.0},
						{StackID: 3, Timestamp: 3.0},
					},
					Stacks: [][]int{
						{0, 1},
						{0, 1},
						{2, 3},
						{2, 3},
					},
				},
				Measurements: json.RawMessage(`{"first_metric":{"unit":"ms","values":[{"elapsed_since_start_ns":100,"value":1},{"elapsed_since_start_ns":200,"value":1.2}]}}`),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			have, err := MergeChunks(test.have)
			if err != nil {
				t.Fatal(err)
			}
			if diff := testutil.Diff(have, test.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
