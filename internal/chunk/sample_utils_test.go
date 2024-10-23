package chunk

import (
	"encoding/json"
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/testutil"
)

func TestMergeSampleChunks(t *testing.T) {
	tests := []struct {
		name  string
		have  []SampleChunk
		want  SampleChunk
		start uint64
		end   uint64
	}{
		{
			name: "contiguous chunks",
			have: []SampleChunk{
				{
					Profile: SampleData{
						Frames: []frame.Frame{
							{Function: "c"},
							{Function: "d"},
						},
						Samples: []Sample{
							{StackID: 0, Timestamp: 3.0},
							{StackID: 1, Timestamp: 4.0},
							{StackID: 1, Timestamp: 5.0}, // outside range, will be dropped
						},
						Stacks: [][]int{
							{0, 1},
							{0, 1},
						},
						ThreadMetadata: map[string]sample.ThreadMetadata{"0x000000016d8fb180": {Name: "com.apple.network.connections"}},
					},
					Measurements: json.RawMessage(`{"first_metric":{"unit":"ms","values":[{"timestamp":2.0,"value":1.2}]}}`),
				},
				// other chunk
				{
					Profile: SampleData{
						Frames: []frame.Frame{
							{Function: "a"},
							{Function: "b"},
						},
						Samples: []Sample{
							{StackID: 0, Timestamp: 0.0},
							{StackID: 0, Timestamp: 1.0},
							{StackID: 1, Timestamp: 2.0},
						},
						Stacks: [][]int{
							{0, 1},
							{0, 1},
						},
						ThreadMetadata: map[string]sample.ThreadMetadata{"0x0000000102adc700": {Name: "com.apple.main-thread"}},
					},
					Measurements: json.RawMessage(`{"first_metric":{"unit":"ms","values":[{"timestamp":1.0,"value":1}]}}`),
				},
			},
			want: SampleChunk{
				Profile: SampleData{
					Frames: []frame.Frame{
						{Function: "a"},
						{Function: "b"},
						{Function: "c"},
						{Function: "d"},
					},
					Samples: []Sample{
						{StackID: 0, Timestamp: 1.0},
						{StackID: 1, Timestamp: 2.0},
						{StackID: 2, Timestamp: 3.0},
						{StackID: 3, Timestamp: 4.0},
					},
					Stacks: [][]int{
						{0, 1},
						{0, 1},
						{2, 3},
						{2, 3},
					},
					ThreadMetadata: map[string]sample.ThreadMetadata{"0x0000000102adc700": {Name: "com.apple.main-thread"}, "0x000000016d8fb180": {Name: "com.apple.network.connections"}},
				},
				Measurements: json.RawMessage(`{"first_metric":{"unit":"ms","values":[{"timestamp":1,"value":1},{"timestamp":2,"value":1.2}]}}`),
			},
			start: uint64(1e9),
			end:   uint64(4e9),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			have, err := MergeSampleChunks(test.have, test.start, test.end)
			if err != nil {
				t.Fatal(err)
			}
			if diff := testutil.Diff(have, test.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
