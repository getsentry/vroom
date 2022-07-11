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
						Fingerprint: 12638153115695167471,
						ID:          12638153115695167471,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 590701660006484780,
								ID:          590701660006484780,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 15673513070074625014,
										ID:          15673513070074625014,
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
						Fingerprint: 12638153115695167471,
						ID:          12638153115695167471,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       1,
								Fingerprint: 590701660006484780,
								ID:          590701660006484780,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 15673513070074625014,
										ID:          15673513070074625014,
										Name:        "symbol2",
									},
								},
							},
						},
					},
					{
						DurationNS:  1,
						EndNS:       2,
						Fingerprint: 12638153115695167468,
						ID:          12638153115695167468,
						Name:        "symbol3",
						StartNS:     1,
						Children: []*nodetree.Node{
							{
								DurationNS:  1,
								EndNS:       2,
								Fingerprint: 590698361471600176,
								ID:          590698361471600176,
								Name:        "symbol4",
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 15670675230562780069,
										ID:          15670675230562780069,
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
						Fingerprint: 12638153115695167471,
						ID:          12638153115695167471,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  2,
								EndNS:       2,
								Fingerprint: 590701660006484780,
								ID:          590701660006484780,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 15673513070074625014,
										ID:          15673513070074625014,
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
						Fingerprint: 12638153115695167471,
						ID:          12638153115695167471,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  2,
								EndNS:       2,
								Fingerprint: 590701660006484780,
								ID:          590701660006484780,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 15673513070074625014,
										ID:          15673513070074625014,
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
						Fingerprint: 12638153115695167471,
						ID:          12638153115695167471,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 590701660006484780,
								ID:          590701660006484780,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       1,
										Fingerprint: 15673513070074625014,
										ID:          15673513070074625014,
										Name:        "symbol2",
									},
									{
										DurationNS:  1,
										EndNS:       3,
										StartNS:     2,
										Fingerprint: 15673513070074625014,
										ID:          15673513070074625014,
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
						Fingerprint: 12638153115695167471,
						ID:          12638153115695167471,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 590701660006484780,
								ID:          590701660006484780,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 15673513070074625014,
										ID:          15673513070074625014,
										Name:        "symbol2",
										StartNS:     1,
									},
									{
										DurationNS:  1,
										EndNS:       4,
										Fingerprint: 15673513070074625014,
										ID:          15673513070074625014,
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
						Fingerprint: 12638153115695167471,
						ID:          12638153115695167471,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  3,
								EndNS:       3,
								Fingerprint: 590701660006484780,
								ID:          590701660006484780,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       2,
										Fingerprint: 15673513070074625014,
										ID:          15673513070074625014,
										Name:        "symbol2",
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       1,
												Fingerprint: 15560027421946782513,
												ID:          15560027421946782513,
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
						Fingerprint: 12638153115695167471,
						ID:          12638153115695167471,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  3,
								EndNS:       4,
								Fingerprint: 590701660006484780,
								ID:          590701660006484780,
								Name:        "symbol1",
								StartNS:     1,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       4,
										Fingerprint: 15673513070074625014,
										ID:          15673513070074625014,
										Name:        "symbol2",
										StartNS:     2,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       4,
												Fingerprint: 15560027421946782513,
												ID:          15560027421946782513,
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
						Fingerprint: 12638153115695167471,
						ID:          12638153115695167471,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 590701660006484780,
								ID:          590701660006484780,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  1,
										EndNS:       2,
										Fingerprint: 15673513070074625014,
										ID:          15673513070074625014,
										Name:        "symbol2",
										StartNS:     1,
									},
									{
										DurationNS:  1,
										EndNS:       3,
										Fingerprint: 15673513070074625015,
										ID:          15673513070074625015,
										Name:        "symbol3",
										StartNS:     2,
									},
									{
										DurationNS:  1,
										EndNS:       4,
										Fingerprint: 15673513070074625008,
										ID:          15673513070074625008,
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
						Fingerprint: 12638153115695167471,
						ID:          12638153115695167471,
						Name:        "symbol0",
						Children: []*nodetree.Node{
							{
								DurationNS:  4,
								EndNS:       4,
								Fingerprint: 590701660006484780,
								ID:          590701660006484780,
								Name:        "symbol1",
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       3,
										Fingerprint: 15673513070074625010,
										ID:          15673513070074625010,
										Name:        "symbol6",
										StartNS:     1,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       3,
												Fingerprint: 15560023023900269569,
												ID:          15560023023900269569,
												Name:        "symbol7",
												StartNS:     2,
											},
										},
									},
									{
										DurationNS:  1,
										EndNS:       4,
										Fingerprint: 15673513070074625009,
										ID:          15673513070074625009,
										Name:        "symbol5",
										StartNS:     3,
										Children: []*nodetree.Node{
											{
												DurationNS:  1,
												EndNS:       4,
												Fingerprint: 15560021924388641460,
												ID:          15560021924388641460,
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
								Fingerprint: 590701660006484783,
								ID:          590701660006484783,
								Name:        "symbol2",
								StartNS:     4,
								Children: []*nodetree.Node{
									{
										DurationNS:  2,
										EndNS:       6,
										Fingerprint: 15673516368609509609,
										ID:          15673516368609509609,
										Name:        "symbol4",
										StartNS:     4,
									},
								},
							},
							{
								DurationNS:  1,
								EndNS:       7,
								Fingerprint: 590701660006484782,
								ID:          590701660006484782,
								Name:        "symbol3",
								StartNS:     6,
							},
						},
					},
					{
						DurationNS:  1,
						EndNS:       8,
						Fingerprint: 12638153115695167463,
						ID:          12638153115695167463,
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
