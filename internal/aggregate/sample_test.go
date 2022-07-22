package aggregate

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/snubautil"
	"github.com/getsentry/vroom/internal/testutil"
)

func TestCallTreeGenerationFromSingleThreadedSamples(t *testing.T) {
	tests := []struct {
		name    string
		profile IosProfile
		want    map[uint64][]*nodetree.Node
	}{
		{
			name: "single root call tree",
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       2,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  2,
						EndNS:       2,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  2,
								EndNS:       2,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  2,
						EndNS:       2,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  2,
								EndNS:       2,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
									},
									{
										DurationNS:  1,
										EndNS:       3,
										StartNS:     2,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										StartNS:     1,
									},
									{
										DurationNS:  1,
										EndNS:       4,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  3,
								EndNS:       3,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       2,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       1,
												Fingerprint: 14019447401716285969,
												Name:        "symbol3",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  3,
								EndNS:       4,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       4,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										StartNS:     2,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       4,
												Fingerprint: 14019447401716285969,
												Name:        "symbol3",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
										StartNS:     1,
									},
									{
										DurationNS:  1,
										EndNS:       3,
										Fingerprint: 16084607411097338727,
										Name:        "symbol3",
										StartNS:     2,
									},
									{
										DurationNS:  1,
										EndNS:       4,
										Fingerprint: 16084607411097338720,
										Name:        "symbol4",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  7,
						EndNS:       7,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       3,
										Fingerprint: 16084607411097338722,
										Name:        "symbol6",
										StartNS:     1,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       3,
												Fingerprint: 3157437670125180841,
												Name:        "symbol7",
												StartNS:     2,
											},
										},
									},
									{
										DurationNS:  1,
										EndNS:       4,
										Fingerprint: 16084607411097338721,
										Name:        "symbol5",
										StartNS:     3,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       4,
												Fingerprint: 18225866209492738612,
												Name:        "symbol7",
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
								StartNS:     4,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       6,
										Fingerprint: 11661425725218473465,
										Name:        "symbol4",
										StartNS:     4,
									},
								},
							},
							{
								DurationNS:  1,
								EndNS:       7,
								Fingerprint: 17905447077897174946,
								Name:        "symbol3",
								StartNS:     6,
							},
						},
					},
					{
						DurationNS:  1,
						EndNS:       8,
						Fingerprint: 1124161485517443919,
						Name:        "symbol8",
						StartNS:     7,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.CallTrees()
			if diff := testutil.Diff(got, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestCallTreeGenerationFromMultiThreadedSamples(t *testing.T) {
	tests := []struct {
		name    string
		profile IosProfile
		want    map[uint64][]*nodetree.Node
	}{
		{
			name: "multiple threads with the same call tree",
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
									},
								},
							},
						},
					},
				},
				2: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
									},
								},
							},
						},
					},
				},
				2: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443908,
						Name:        "symbol3",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       2,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
										StartNS:     1,
									},
								},
							},
						},
					},
				},
				2: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       2,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       2,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
										StartNS:     1,
									},
								},
							},
						},
					},
				},
				2: []*nodetree.Node{
					{
						DurationNS:  5,
						EndNS:       5,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  5,
								EndNS:       5,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  5,
										EndNS:       5,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
									},
								},
							},
						},
					},
					{
						DurationNS:  1,
						EndNS:       6,
						Fingerprint: 1124161485517443908,
						Name:        "symbol3",
						StartNS:     5,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       6,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								StartNS:     5,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       6,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
										StartNS:     5,
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       2,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
										StartNS:     1,
									},
								},
							},
						},
					},
				},
				2: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       2,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
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
			profile: IosProfile{
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
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
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
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  2,
								EndNS:       3,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       3,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
										StartNS:     1,
									},
								},
							},
						},
					},
				},
				2: []*nodetree.Node{
					{
						DurationNS:  2,
						EndNS:       2,
						Fingerprint: 1124161485517443911,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  2,
								EndNS:       2,
								Fingerprint: 17905447077897174944,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       2,
										Fingerprint: 16084607411097338726,
										Name:        "symbol2",
									},
								},
							},
						},
					},
					{
						DurationNS:  2,
						EndNS:       4,
						Fingerprint: 1124161485517443908,
						Name:        "symbol3",
						StartNS:     2,
						Children: []*nodetree.Node{
							{
								DurationNS:  2,
								EndNS:       4,
								Fingerprint: 7967440964543288636,
								Name:        "symbol4",
								StartNS:     2,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       4,
										Fingerprint: 13274796176329250277,
										Name:        "symbol5",
										StartNS:     2,
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
			got := tt.profile.CallTrees()
			if diff := testutil.Diff(got, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func BenchmarkCallTrees(b *testing.B) {
	var p snubautil.Profile
	f, err := os.Open("../../test/data/cocoa.json")
	if err != nil {
		b.Fatal(err)
	}
	err = json.NewDecoder(f).Decode(&p)
	if err != nil {
		b.Fatal(err)
	}
	var profile IosProfile
	err = json.Unmarshal([]byte(p.Profile), &profile)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	var total int
	for n := 0; n < b.N; n++ {
		c := profile.CallTrees()
		total += len(c)
	}
	b.Logf("Total call trees generated: %d", total)
}
