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

func TestMergeSampleChunksAttachments(t *testing.T) {
	chunks := []SampleChunk{
		{
			Attachments: []Attachment{
				{Name: "raw_profile", ContentType: "application/x-perfetto", StoredID: "raw_profile_b"},
				// Attachments without a stored ID are skipped.
				{Name: "raw_profile", ContentType: "application/x-perfetto", StoredID: ""},
			},
			Profile: SampleData{
				Frames: []frame.Frame{
					{Function: "b"},
				},
				Samples: []Sample{
					{StackID: 0, Timestamp: 3.0},
					{StackID: 0, Timestamp: 4.0},
				},
				Stacks: [][]int{
					{0},
				},
			},
		},
		// chunk without attachments
		{
			Profile: SampleData{
				Frames: []frame.Frame{
					{Function: "c"},
				},
				Samples: []Sample{
					{StackID: 0, Timestamp: 5.0},
					{StackID: 0, Timestamp: 6.0},
				},
				Stacks: [][]int{
					{0},
				},
			},
		},
		{
			Attachments: []Attachment{
				{Name: "raw_profile", ContentType: "application/x-perfetto", StoredID: "raw_profile_a"},
			},
			Profile: SampleData{
				Frames: []frame.Frame{
					{Function: "a"},
				},
				Samples: []Sample{
					{StackID: 0, Timestamp: 1.0},
					{StackID: 0, Timestamp: 2.0},
				},
				Stacks: [][]int{
					{0},
				},
			},
		},
	}

	merged, err := MergeSampleChunks(chunks, 0, uint64(10e9))
	if err != nil {
		t.Fatal(err)
	}

	// Attachments follow the merged chunk order. Note: the chunk sort in
	// MergeSampleChunks is only well-defined for non-overlapping chunks,
	// so no strict chronological order is guaranteed.
	want := []Attachment{
		{Name: "raw_profile", ContentType: "application/x-perfetto", StoredID: "raw_profile_a"},
		{Name: "raw_profile", ContentType: "application/x-perfetto", StoredID: "raw_profile_b"},
	}
	if diff := testutil.Diff(merged.Attachments, want); diff != "" {
		t.Fatalf("Result mismatch: got - want +\n%s", diff)
	}
}
