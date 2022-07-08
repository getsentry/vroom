package aggregate

import (
	"testing"

	"github.com/getsentry/vroom/internal/nodetree"
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
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 14342200307895820206,
						ID:          14342200307895820206,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 15449812301751522459,
								ID:          15449812301751522459,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 12688607006961369939,
										ID:          12688607006961369939,
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
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "5", Symbol: "symbol5"},
							{InstructionAddr: "4", Symbol: "symbol4"},
							{InstructionAddr: "3", Symbol: "symbol3"},
						},
					},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  1,
						EndNS:       1,
						Fingerprint: 14342200307895820206,
						ID:          14342200307895820206,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 15449812301751522459,
								ID:          15449812301751522459,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 12688607006961369939,
										ID:          12688607006961369939,
										Name:        "symbol2",
									},
								},
							},
						},
					},
					{
						DurationNS:  1,
						EndNS:       2,
						Fingerprint: 14342200307895820205,
						ID:          14342200307895820205,
						Name:        "symbol3",
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       2,
								Fingerprint: 15449811202239894211,
								ID:          15449811202239894211,
								Name:        "symbol4",
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 12687609749914771820,
										ID:          12687609749914771820,
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
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  2,
						EndNS:       2,
						Fingerprint: 14342200307895820206,
						ID:          14342200307895820206,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  2,
								EndNS:       2,
								Fingerprint: 15449812301751522459,
								ID:          15449812301751522459,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 12688607006961369939,
										ID:          12688607006961369939,
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
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  2,
						EndNS:       2,
						Fingerprint: 14342200307895820206,
						ID:          14342200307895820206,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  2,
								EndNS:       2,
								Fingerprint: 15449812301751522459,
								ID:          15449812301751522459,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 12688607006961369939,
										ID:          12688607006961369939,
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
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 14342200307895820206,
						ID:          14342200307895820206,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 15449812301751522459,
								ID:          15449812301751522459,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 12688607006961369939,
										ID:          12688607006961369939,
										Name:        "symbol2",
									},
									{
										DurationNS:  1,
										EndNS:       3,
										StartNS:     2,
										Fingerprint: 12688607006961369939,
										ID:          12688607006961369939,
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
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 14342200307895820206,
						ID:          14342200307895820206,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 15449812301751522459,
								ID:          15449812301751522459,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 12688607006961369939,
										ID:          12688607006961369939,
										Name:        "symbol2",
										StartNS:     1,
									},
									{
										DurationNS:  1,
										EndNS:       4,
										Fingerprint: 12688607006961369939,
										ID:          12688607006961369939,
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
							{InstructionAddr: "3", Symbol: "symbol3"},
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 14342200307895820206,
						ID:          14342200307895820206,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  3,
								EndNS:       3,
								Fingerprint: 15449812301751522459,
								ID:          15449812301751522459,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       2,
										Fingerprint: 12688607006961369939,
										ID:          12688607006961369939,
										Name:        "symbol2",
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       1,
												Fingerprint: 12543359363948652090,
												ID:          12543359363948652090,
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
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "3", Symbol: "symbol3"},
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 14342200307895820206,
						ID:          14342200307895820206,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  3,
								EndNS:       4,
								Fingerprint: 15449812301751522459,
								ID:          15449812301751522459,
								Name:        "symbol1",
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       4,
										Fingerprint: 12688607006961369939,
										ID:          12688607006961369939,
										Name:        "symbol2",
										StartNS:     2,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       4,
												Fingerprint: 12543359363948652090,
												ID:          12543359363948652090,
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
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "3", Symbol: "symbol3"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "4", Symbol: "symbol4"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  4,
						EndNS:       4,
						Fingerprint: 14342200307895820206,
						ID:          14342200307895820206,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 15449812301751522459,
								ID:          15449812301751522459,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 12688607006961369939,
										ID:          12688607006961369939,
										Name:        "symbol2",
										StartNS:     1,
									},
									{
										DurationNS:  1,
										EndNS:       3,
										Fingerprint: 12688607006961369938,
										ID:          12688607006961369938,
										Name:        "symbol3",
										StartNS:     2,
									},
									{
										DurationNS:  1,
										EndNS:       4,
										Fingerprint: 12688607006961369941,
										ID:          12688607006961369941,
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
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 2,
						Frames: []IosFrame{
							{InstructionAddr: "6", Symbol: "symbol6"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 3,
						Frames: []IosFrame{
							{InstructionAddr: "7", Symbol: "symbol7"},
							{InstructionAddr: "6", Symbol: "symbol6"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 4,
						Frames: []IosFrame{
							{InstructionAddr: "7", Symbol: "symbol7"},
							{InstructionAddr: "5", Symbol: "symbol5"},
							{InstructionAddr: "1", Symbol: "symbol1"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 5,
						Frames: []IosFrame{
							{InstructionAddr: "4", Symbol: "symbol4"},
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 6,
						Frames: []IosFrame{
							{InstructionAddr: "4", Symbol: "symbol4"},
							{InstructionAddr: "2", Symbol: "symbol2"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 7,
						Frames: []IosFrame{
							{InstructionAddr: "3", Symbol: "symbol3"},
							{InstructionAddr: "0", Symbol: "symbol0"},
						},
					},
					{
						ThreadID:            1,
						RelativeTimestampNS: 8,
						Frames: []IosFrame{
							{InstructionAddr: "8", Symbol: "symbol8"},
						},
					},
				},
			},
			want: map[uint64][]*nodetree.Node{
				1: []*nodetree.Node{
					{
						DurationNS:  7,
						EndNS:       7,
						Fingerprint: 14342200307895820206,
						ID:          14342200307895820206,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 15449812301751522459,
								ID:          15449812301751522459,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       3,
										Fingerprint: 12688607006961369943,
										ID:          12688607006961369943,
										Name:        "symbol6",
										StartNS:     1,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       3,
												Fingerprint: 12543363761995164898,
												ID:          12543363761995164898,
												Name:        "symbol7",
												StartNS:     2,
											},
										},
									},
									{
										DurationNS:  1,
										EndNS:       4,
										Fingerprint: 12688607006961369940,
										ID:          12688607006961369940,
										Name:        "symbol5",
										StartNS:     3,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       4,
												Fingerprint: 12543360463460280203,
												ID:          12543360463460280203,
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
								Fingerprint: 15449812301751522456,
								ID:          15449812301751522456,
								Name:        "symbol2",
								StartNS:     4,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       6,
										Fingerprint: 12688603708426485372,
										ID:          12688603708426485372,
										Name:        "symbol4",
										StartNS:     4,
									},
								},
							},
							{
								DurationNS:  1,
								EndNS:       7,
								Fingerprint: 15449812301751522457,
								ID:          15449812301751522457,
								Name:        "symbol3",
								StartNS:     6,
							},
						},
					},
					{
						DurationNS:  1,
						EndNS:       8,
						Fingerprint: 14342200307895820198,
						ID:          14342200307895820198,
						Name:        "symbol8",
						StartNS:     7,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.CallTrees()
			if diff := testutil.Diff(got, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
