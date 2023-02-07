package profile

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/testutil"
)

func TestCallTreeGenerationFromSingleThreadedSamples(t *testing.T) {
	tests := []struct {
		name    string
		profile IOS
		want    map[uint64][]*nodetree.Node
	}{
		{
			name: "single root call tree",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple root call trees",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "5", Function: "symbol5"},
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "3", Function: "symbol3"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
									},
								},
							},
						},
					},
					{
						DurationNS:  1,
						EndNS:       2,
						Fingerprint: 1124161485517443908,
						Name:        "symbol3",
						SampleCount: 1,
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       2,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								SampleCount: 1,
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
										SampleCount: 1,
										StartNS:     1,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "single root call tree with disappearing leaf",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  2,
						EndNS:       2,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 2,
						Children: []*nodetree.Node{
							{
								DurationNS:  2,
								EndNS:       2,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 2,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "single root call tree with appearing leaf",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  2,
						EndNS:       2,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 2,
						Children: []*nodetree.Node{
							{
								DurationNS:  2,
								EndNS:       2,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 2,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
										StartNS:     1,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "single root call tree with repeated disappearing leaf",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 4,
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 4,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
									},
									{
										DurationNS:  1,
										EndNS:       3,
										StartNS:     2,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "single root call tree with repeated appearing leaf",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 4,
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 4,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
										StartNS:     1,
									},
									{
										DurationNS:  1,
										EndNS:       4,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
										StartNS:     3,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "single root call tree with disappearing leaves",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "3", Function: "symbol3"},
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 4,
						Children: []*nodetree.Node{
							{
								DurationNS:  3,
								EndNS:       3,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 3,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       2,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 2,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       1,
												Fingerprint: 14019447401716285969,
												Name:        "symbol3",
												SampleCount: 1,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "single root call tree with appearing leaves",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "3", Function: "symbol3"},
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 4,
						Children: []*nodetree.Node{
							{
								DurationNS:  3,
								EndNS:       4,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 3,
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       4,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 2,
										StartNS:     2,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       4,
												Fingerprint: 14019447401716285969,
												Name:        "symbol3",
												SampleCount: 1,
												StartNS:     3,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "single root call tree with multiple unique leaves",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "3", Function: "symbol3"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 4,
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 4,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
										StartNS:     1,
									},
									{
										DurationNS:  1,
										EndNS:       3,
										Fingerprint: 16084607411097338727,
										Name:        "symbol3",
										SampleCount: 1,
										StartNS:     2,
									},
									{
										DurationNS:  1,
										EndNS:       4,
										Fingerprint: 16084607411097338720,
										Name:        "symbol4",
										SampleCount: 1,
										StartNS:     3,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "single root call tree with multiple unique leaves at different levels",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "6", Function: "symbol6"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "7", Function: "symbol7"},
							{InstructionAddr: "6", Function: "symbol6"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "7", Function: "symbol7"},
							{InstructionAddr: "5", Function: "symbol5"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 5,
						Frames: []IosFrame{
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 6,
						Frames: []IosFrame{
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 7,
						Frames: []IosFrame{
							{InstructionAddr: "3", Function: "symbol3"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 8,
						Frames: []IosFrame{
							{InstructionAddr: "8", Function: "symbol8"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  7,
						EndNS:       7,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 7,
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 4,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       3,
										Fingerprint: 16084607411097338722,
										Name:        "symbol6",
										SampleCount: 2,
										StartNS:     1,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       3,
												Fingerprint: 3157437670125180841,
												Name:        "symbol7",
												SampleCount: 1,
												StartNS:     2,
											},
										},
									},
									{
										DurationNS:  1,
										EndNS:       4,
										Fingerprint: 16084607411097338721,
										Name:        "symbol5",
										SampleCount: 1,
										StartNS:     3,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       4,
												Fingerprint: 18225866209492738612,
												Name:        "symbol7",
												SampleCount: 1,
												StartNS:     3,
											},
										},
									},
								},
							},
							{
								DurationNS:  2,
								EndNS:       6,
								Fingerprint: 17905447077897174947,
								Name:        "symbol2",
								SampleCount: 2,
								StartNS:     4,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       6,
										Fingerprint: 11661425725218473465,
										Name:        "symbol4",
										SampleCount: 2,
										StartNS:     4,
									},
								},
							},
							{
								DurationNS:  1,
								EndNS:       7,
								Fingerprint: 17905447077897174946,
								Name:        "symbol3",
								SampleCount: 1,
								StartNS:     6,
							},
						},
					},
					{
						DurationNS:  1,
						EndNS:       8,
						Fingerprint: 1124161485517443919,
						Name:        "symbol8",
						SampleCount: 1,
						StartNS:     7,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.CallTrees(false)
			if diff := testutil.Diff(got, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestCallTreeGenerationFromMultiThreadedSamples(t *testing.T) {
	tests := []struct {
		name    string
		profile IOS
		want    map[uint64][]*nodetree.Node
	}{
		{
			name: "multiple threads with the same call tree",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            2,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple threads with different call trees",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            2,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "5", Function: "symbol5"},
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "3", Function: "symbol3"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple threads with sequential samples",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "5", Function: "symbol5"},
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "3", Function: "symbol3"},
						},
					},
					{
						ThreadID:            2,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            2,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "5", Function: "symbol5"},
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "3", Function: "symbol3"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
									},
								},
							},
						},
					},
					{
						DurationNS:  1,
						EndNS:       2,
						Fingerprint: 1124161485517443908,
						Name:        "symbol3",
						SampleCount: 1,
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       2,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								SampleCount: 1,
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
										SampleCount: 1,
										StartNS:     1,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple threads with non-sequential samples",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "5", Function: "symbol5"},
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "3", Function: "symbol3"},
						},
					},
					{
						ThreadID:            2,
						RelativeTimestampNS: 5,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            2,
						RelativeTimestampNS: 6,
						Frames: []IosFrame{
							{InstructionAddr: "5", Function: "symbol5"},
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "3", Function: "symbol3"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
									},
								},
							},
						},
					},
					{
						DurationNS:  1,
						EndNS:       2,
						Fingerprint: 1124161485517443908,
						Name:        "symbol3",
						SampleCount: 1,
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       2,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								SampleCount: 1,
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
										SampleCount: 1,
										StartNS:     1,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple threads with interleaved sequential samples",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            2,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "5", Function: "symbol5"},
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "3", Function: "symbol3"},
						},
					},

					{
						ThreadID:            2,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "5", Function: "symbol5"},
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "3", Function: "symbol3"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
									},
								},
							},
						},
					},
					{
						DurationNS:  1,
						EndNS:       2,
						Fingerprint: 1124161485517443908,
						Name:        "symbol3",
						SampleCount: 1,
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       2,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								SampleCount: 1,
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
										SampleCount: 1,
										StartNS:     1,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple threads with interleaved non-sequential samples",
			profile: IOS{
				Samples: []Sample{
					{
						ThreadID:            1,
						RelativeTimestampNS: 1,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            2,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "2", Function: "symbol2"},
							{InstructionAddr: "1", Function: "symbol1"},
							{InstructionAddr: "0", Function: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "5", Function: "symbol5"},
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "3", Function: "symbol3"},
						},
					},
					{
						ThreadID:            2,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "5", Function: "symbol5"},
							{InstructionAddr: "4", Function: "symbol4"},
							{InstructionAddr: "3", Function: "symbol3"},
						},
					},
				},
				ThreadMetadata: map[string]ThreadMetadata{
					"1": {IsMain: true},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						SampleCount: 1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								SampleCount: 1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										SampleCount: 1,
									},
								},
							},
						},
					},
					{
						DurationNS:  2,
						EndNS:       3,
						Fingerprint: 1124161485517443908,
						Name:        "symbol3",
						SampleCount: 1,
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  2,
								EndNS:       3,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								SampleCount: 1,
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       3,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
										SampleCount: 1,
										StartNS:     1,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.CallTrees(false)
			if diff := testutil.Diff(got, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestCallTreeGenerationFromMultipleProfiles(t *testing.T) {
	p1 := IOS{
		Samples: []Sample{
			{
				ThreadID:            1,
				RelativeTimestampNS: 1,
				Frames: []IosFrame{
					{InstructionAddr: "2", Function: "symbol2", Package: "/home/pierre/release1/package1"},
					{InstructionAddr: "1", Function: "symbol1", Package: "/home/pierre/release1/package1"},
					{InstructionAddr: "0", Function: "symbol0", Package: "/home/pierre/release1/package1"},
				},
			},
			{
				ThreadID:            2,
				RelativeTimestampNS: 1,
				Frames: []IosFrame{
					{InstructionAddr: "2", Function: "symbol2", Package: "/home/pierre/release1/package1"},
					{InstructionAddr: "1", Function: "symbol1", Package: "/home/pierre/release1/package1"},
					{InstructionAddr: "0", Function: "symbol0", Package: "/home/pierre/release1/package1"},
				},
			},
		},
	}
	p2 := IOS{
		Samples: []Sample{
			{
				ThreadID:            1,
				RelativeTimestampNS: 1,
				Frames: []IosFrame{
					{InstructionAddr: "2", Function: "symbol2", Package: "/home/pierre/release2/package1"},
					{InstructionAddr: "1", Function: "symbol1", Package: "/home/pierre/release2/package1"},
					{InstructionAddr: "0", Function: "symbol0", Package: "/home/pierre/release2/package1"},
				},
			},
			{
				ThreadID:            2,
				RelativeTimestampNS: 1,
				Frames: []IosFrame{
					{InstructionAddr: "2", Function: "symbol2", Package: "/home/pierre/release2/package1"},
					{InstructionAddr: "1", Function: "symbol1", Package: "/home/pierre/release2/package1"},
					{InstructionAddr: "0", Function: "symbol0", Package: "/home/pierre/release2/package1"},
				},
			},
		},
	}

	if diff := testutil.Diff(p1.CallTrees(false), p2.CallTrees(false)); diff != "" {
		t.Fatalf("Result mismatch: got - want +\n%s", diff)
	}
}

func BenchmarkCallTrees(b *testing.B) {
	var p LegacyProfile
	f, err := os.Open("../../test/data/cocoa.json")
	if err != nil {
		b.Fatal(err)
	}
	err = json.NewDecoder(f).Decode(&p)
	if err != nil {
		b.Fatal(err)
	}
	var iosProfile IOS
	err = json.Unmarshal([]byte(p.Profile), &iosProfile)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	var total int
	for n := 0; n < b.N; n++ {
		c := iosProfile.CallTrees(false)
		total += len(c)
	}
	b.Logf("Total call trees generated: %d", total)
}
