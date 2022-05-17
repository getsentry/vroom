package calltree

import (
	"testing"

	"github.com/getsentry/vroom/internal/testutil"
)

func TestSingleThreadBacktracePAggregation(t *testing.T) {
	type bt struct {
		timestampNs uint64
		addresses   []string
	}

	tests := []struct {
		name       string
		backtraces []bt
		want       []*CallTreeP
	}{
		{
			name: "single root call tree",
			backtraces: []bt{
				{0, []string{"2", "1", "0"}},
			},
			want: []*CallTreeP{
				{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
					{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"2", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", nil},
					}},
				}},
			},
		},
		{
			name: "multiple root call trees",
			backtraces: []bt{
				{0, []string{"2", "1", "0"}},
				{1, []string{"5", "4", "3"}},
			},
			want: []*CallTreeP{
				{"0", 0, 0, 1, 0, "trace1", "", "", []*CallTreeP{
					{"1", 0, 0, 1, 0, "trace1", "", "", []*CallTreeP{
						{"2", 0, 0, 1, 1, "trace1", "", "", nil},
					}},
				}},
				{"3", 0, 1, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
					{"4", 0, 1, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"5", 0, 1, NoEndTime, NoEndTime, "trace1", "", "", nil},
					}},
				}},
			},
		},
		{
			name: "single root call tree with disappearing leaf",
			backtraces: []bt{
				{0, []string{"2", "1", "0"}},
				{10, []string{"1", "0"}},
			},
			want: []*CallTreeP{
				{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
					{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"2", 0, 0, 10, 10, "trace1", "", "", nil},
					}},
				}},
			},
		},
		{
			name: "single root call tree with appearing leaf",
			backtraces: []bt{
				{0, []string{"1", "0"}},
				{10, []string{"2", "1", "0"}},
			},
			want: []*CallTreeP{
				{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
					{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"2", 0, 10, NoEndTime, NoEndTime, "trace1", "", "", nil},
					}},
				}},
			},
		},
		{
			name: "single root call tree with repeated disappearing leaf",
			backtraces: []bt{
				{0, []string{"2", "1", "0"}},
				{10, []string{"1", "0"}},
				{20, []string{"2", "1", "0"}},
				{30, []string{"1", "0"}},
			},
			want: []*CallTreeP{
				{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
					{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"2", 0, 0, 10, 10, "trace1", "", "", nil},
						{"2", 0, 20, 30, 10, "trace1", "", "", nil},
					}},
				}},
			},
		},
		{
			name: "single root call tree with repeated appearing leaf",
			backtraces: []bt{
				{0, []string{"1", "0"}},
				{10, []string{"2", "1", "0"}},
				{20, []string{"1", "0"}},
				{30, []string{"2", "1", "0"}},
			},
			want: []*CallTreeP{
				{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
					{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"2", 0, 10, 20, 10, "trace1", "", "", nil},
						{"2", 0, 30, NoEndTime, NoEndTime, "trace1", "", "", nil},
					}},
				}},
			},
		},
		{
			name: "single root call tree with disappearing leaves",
			backtraces: []bt{
				{0, []string{"2", "1", "0"}},
				{10, []string{"1", "0"}},
				{20, []string{"0"}},
			},
			want: []*CallTreeP{
				{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
					{"1", 0, 0, 20, 10, "trace1", "", "", []*CallTreeP{
						{"2", 0, 0, 10, 10, "trace1", "", "", nil},
					}},
				}},
			},
		},
		{
			name: "single root call tree with appearing leaves",
			backtraces: []bt{
				{0, []string{"0"}},
				{10, []string{"1", "0"}},
				{20, []string{"2", "1", "0"}},
			},
			want: []*CallTreeP{
				{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
					{"1", 0, 10, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"2", 0, 20, NoEndTime, NoEndTime, "trace1", "", "", nil},
					}},
				}},
			},
		},
		{
			name: "single root call tree with multiple unique leaves",
			backtraces: []bt{
				{0, []string{"1", "0"}},
				{10, []string{"2", "1", "0"}},
				{20, []string{"3", "1", "0"}},
				{30, []string{"4", "1", "0"}},
			},
			want: []*CallTreeP{
				{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
					{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"2", 0, 10, 20, 10, "trace1", "", "", nil},
						{"3", 0, 20, 30, 10, "trace1", "", "", nil},
						{"4", 0, 30, NoEndTime, NoEndTime, "trace1", "", "", nil},
					}},
				}},
			},
		},
		{
			name: "single root call tree with multiple unique leaves at different levels",
			backtraces: []bt{
				{0, []string{"1", "0"}},
				{10, []string{"6", "1", "0"}},
				{20, []string{"7", "6", "1", "0"}},
				{30, []string{"7", "5", "1", "0"}},
				{40, []string{"4", "2", "0"}},
				{50, []string{"5", "2", "0"}},
				{60, []string{"3", "0"}},
				{70, []string{"8"}},
			},
			want: []*CallTreeP{
				{"0", 0, 0, 70, 0, "trace1", "", "", []*CallTreeP{
					{"1", 0, 0, 40, 10, "trace1", "", "", []*CallTreeP{
						{"6", 0, 10, 30, 10, "trace1", "", "", []*CallTreeP{
							{"7", 0, 20, 30, 10, "trace1", "", "", nil},
						}},
						{"5", 0, 30, 40, 0, "trace1", "", "", []*CallTreeP{
							{"7", 0, 30, 40, 10, "trace1", "", "", nil},
						}},
					}},
					{"2", 0, 40, 60, 0, "trace1", "", "", []*CallTreeP{
						{"4", 0, 40, 50, 10, "trace1", "", "", nil},
						{"5", 0, 50, 60, 10, "trace1", "", "", nil},
					}},
					{"3", 0, 60, 70, 10, "trace1", "", "", nil},
				}},
				{"8", 0, 70, NoEndTime, NoEndTime, "trace1", "", "", nil},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			agg := NewBacktraceAggregatorP()
			for _, backtrace := range tt.backtraces {
				agg.Update(BacktraceP{
					ProfileID:   "trace1",
					TimestampNs: backtrace.timestampNs,
					Addresses:   backtrace.addresses,
				})
			}
			agg.Finalize()
			got := agg.ProfileIDToCallTreeInfo["trace1"][0]
			if diff := testutil.Diff(got, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestMultiThreadBacktracePAggregation(t *testing.T) {
	type bt struct {
		threadID    uint64
		timestampNs uint64
		addresses   []string
	}

	tests := []struct {
		name       string
		backtraces []bt
		want       map[uint64][]*CallTreeP
	}{
		{
			name: "multiple threads with same call tree",
			backtraces: []bt{
				{0, 0, []string{"2", "1", "0"}},
				{1, 0, []string{"2", "1", "0"}},
			},
			want: map[uint64][]*CallTreeP{
				0: []*CallTreeP{
					{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"2", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", nil},
						}},
					}},
				},
				1: []*CallTreeP{
					{"0", 1, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"1", 1, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"2", 1, 0, NoEndTime, NoEndTime, "trace1", "", "", nil},
						}},
					}},
				},
			},
		},
		{
			name: "multiple threads with different call trees",
			backtraces: []bt{
				{0, 0, []string{"2", "1", "0"}},
				{1, 0, []string{"5", "4", "3"}},
			},
			want: map[uint64][]*CallTreeP{
				0: []*CallTreeP{
					{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"2", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", nil},
						}},
					}},
				},
				1: []*CallTreeP{
					{"3", 1, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"4", 1, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"5", 1, 0, NoEndTime, NoEndTime, "trace1", "", "", nil},
						}},
					}},
				},
			},
		},
		{
			name: "sequential thread samples",
			backtraces: []bt{
				{0, 0, []string{"2", "1", "0"}},
				{0, 10, []string{"1", "0"}},
				{1, 0, []string{"5", "4", "3"}},
				{1, 10, []string{"4", "3"}},
			},
			want: map[uint64][]*CallTreeP{
				0: []*CallTreeP{
					{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"2", 0, 0, 10, 10, "trace1", "", "", nil},
						}},
					}},
				},
				1: []*CallTreeP{
					{"3", 1, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"4", 1, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"5", 1, 0, 10, 10, "trace1", "", "", nil},
						}},
					}},
				},
			},
		},
		{
			name: "sequential thread samples with different timestamps",
			backtraces: []bt{
				{0, 0, []string{"2", "1", "0"}},
				{0, 10, []string{"1", "0"}},
				{1, 20, []string{"5", "4", "3"}},
				{1, 30, []string{"4", "3"}},
			},
			want: map[uint64][]*CallTreeP{
				0: []*CallTreeP{
					{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"2", 0, 0, 10, 10, "trace1", "", "", nil},
						}},
					}},
				},
				1: []*CallTreeP{
					{"3", 1, 20, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"4", 1, 20, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"5", 1, 20, 30, 10, "trace1", "", "", nil},
						}},
					}},
				},
			},
		},
		{
			name: "interleaved thread samples",
			backtraces: []bt{
				{0, 0, []string{"2", "1", "0"}},
				{1, 0, []string{"5", "4", "3"}},
				{0, 10, []string{"1", "0"}},
				{1, 10, []string{"4", "3"}},
			},
			want: map[uint64][]*CallTreeP{
				0: []*CallTreeP{
					{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"2", 0, 0, 10, 10, "trace1", "", "", nil},
						}},
					}},
				},
				1: []*CallTreeP{
					{"3", 1, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"4", 1, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"5", 1, 0, 10, 10, "trace1", "", "", nil},
						}},
					}},
				},
			},
		},
		{
			name: "interleaved thread samples with different timestamps",
			backtraces: []bt{
				{0, 0, []string{"2", "1", "0"}},
				{1, 40, []string{"5", "4", "3"}},
				{0, 10, []string{"1", "0"}},
				{1, 50, []string{"4", "3"}},
			},
			want: map[uint64][]*CallTreeP{
				0: []*CallTreeP{
					{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"2", 0, 0, 10, 10, "trace1", "", "", nil},
						}},
					}},
				},
				1: []*CallTreeP{
					{"3", 1, 40, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
						{"4", 1, 40, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"5", 1, 40, 50, 10, "trace1", "", "", nil},
						}},
					}},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			agg := NewBacktraceAggregatorP()
			for _, backtrace := range tt.backtraces {
				agg.Update(BacktraceP{
					ProfileID:   "trace1",
					ThreadID:    backtrace.threadID,
					TimestampNs: backtrace.timestampNs,
					Addresses:   backtrace.addresses,
				})
			}
			agg.Finalize()
			got := agg.ProfileIDToCallTreeInfo["trace1"]
			if diff := testutil.Diff(got, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestMultiTraceBacktracePAggregation(t *testing.T) {
	type bt struct {
		profileID   string
		threadID    uint64
		timestampNs uint64
		addresses   []string
	}

	tests := []struct {
		name       string
		backtraces []bt
		want       map[string]map[uint64][]*CallTreeP
	}{
		{
			name: "multiple traces with same thread and same call tree",
			backtraces: []bt{
				{"trace1", 0, 0, []string{"2", "1", "0"}},
				{"trace2", 0, 0, []string{"2", "1", "0"}},
			},
			want: map[string]map[uint64][]*CallTreeP{
				"trace1": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
								{"2", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", nil},
							}},
						}},
					},
				},
				"trace2": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"0", 0, 0, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
							{"1", 0, 0, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
								{"2", 0, 0, NoEndTime, NoEndTime, "trace2", "", "", nil},
							}},
						}},
					},
				},
			},
		},
		{
			name: "multiple traces with same thread and different call trees",
			backtraces: []bt{
				{"trace1", 0, 0, []string{"2", "1", "0"}},
				{"trace2", 0, 0, []string{"5", "4", "3"}},
			},
			want: map[string]map[uint64][]*CallTreeP{
				"trace1": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
								{"2", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", nil},
							}},
						}},
					},
				},
				"trace2": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"3", 0, 0, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
							{"4", 0, 0, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
								{"5", 0, 0, NoEndTime, NoEndTime, "trace2", "", "", nil},
							}},
						}},
					},
				},
			},
		},
		{
			name: "multiple traces with different thread and same call trees",
			backtraces: []bt{
				{"trace1", 0, 0, []string{"2", "1", "0"}},
				{"trace2", 1, 0, []string{"2", "1", "0"}},
			},
			want: map[string]map[uint64][]*CallTreeP{
				"trace1": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
								{"2", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", nil},
							}},
						}},
					},
				},
				"trace2": map[uint64][]*CallTreeP{
					1: []*CallTreeP{
						{"0", 1, 0, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
							{"1", 1, 0, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
								{"2", 1, 0, NoEndTime, NoEndTime, "trace2", "", "", nil},
							}},
						}},
					},
				},
			},
		},
		{
			name: "multiple traces with different thread and different call trees",
			backtraces: []bt{
				{"trace1", 0, 0, []string{"2", "1", "0"}},
				{"trace2", 1, 0, []string{"5", "4", "3"}},
			},
			want: map[string]map[uint64][]*CallTreeP{
				"trace1": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
								{"2", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", nil},
							}},
						}},
					},
				},
				"trace2": map[uint64][]*CallTreeP{
					1: []*CallTreeP{
						{"3", 1, 0, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
							{"4", 1, 0, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
								{"5", 1, 0, NoEndTime, NoEndTime, "trace2", "", "", nil},
							}},
						}},
					},
				},
			},
		},
		{
			name: "sequential traces",
			backtraces: []bt{
				{"trace1", 0, 0, []string{"2", "1", "0"}},
				{"trace1", 0, 10, []string{"1", "0"}},
				{"trace2", 0, 0, []string{"5", "4", "3"}},
				{"trace2", 0, 10, []string{"4", "3"}},
			},
			want: map[string]map[uint64][]*CallTreeP{
				"trace1": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
								{"2", 0, 0, 10, 10, "trace1", "", "", nil},
							}},
						}},
					},
				},
				"trace2": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"3", 0, 0, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
							{"4", 0, 0, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
								{"5", 0, 0, 10, 10, "trace2", "", "", nil},
							}},
						}},
					},
				},
			},
		},
		{
			name: "sequential traces with different timestamps",
			backtraces: []bt{
				{"trace1", 0, 0, []string{"2", "1", "0"}},
				{"trace1", 0, 10, []string{"1", "0"}},
				{"trace2", 0, 20, []string{"5", "4", "3"}},
				{"trace2", 0, 30, []string{"4", "3"}},
			},
			want: map[string]map[uint64][]*CallTreeP{
				"trace1": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"0", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"1", 0, 0, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
								{"2", 0, 0, 10, 10, "trace1", "", "", nil},
							}},
						}},
					},
				},
				"trace2": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"3", 0, 20, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
							{"4", 0, 20, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
								{"5", 0, 20, 30, 10, "trace2", "", "", nil},
							}},
						}},
					},
				},
			},
		},
		{
			name: "interleaved traces",
			backtraces: []bt{
				{"trace1", 0, 0, []string{"2", "1", "0"}},
				{"trace2", 0, 0, []string{"5", "4", "3"}},
				{"trace1", 0, 10, []string{"1", "0"}},
				{"trace2", 0, 10, []string{"4", "3"}},
			},
			want: map[string]map[uint64][]*CallTreeP{
				"trace1": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"0", 0, 10, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"1", 0, 10, NoEndTime, NoEndTime, "trace1", "", "", nil},
						}},
					},
				},
				"trace2": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"3", 0, 10, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
							{"4", 0, 10, NoEndTime, NoEndTime, "trace2", "", "", nil},
						}},
					},
				},
			},
		},
		{
			name: "interleaved traces with different timestamps",
			backtraces: []bt{
				{"trace1", 0, 0, []string{"2", "1", "0"}},
				{"trace2", 0, 20, []string{"5", "4", "3"}},
				{"trace1", 0, 10, []string{"1", "0"}},
				{"trace2", 0, 30, []string{"4", "3"}},
			},
			want: map[string]map[uint64][]*CallTreeP{
				"trace1": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"0", 0, 10, NoEndTime, NoEndTime, "trace1", "", "", []*CallTreeP{
							{"1", 0, 10, NoEndTime, NoEndTime, "trace1", "", "", nil},
						}},
					},
				},
				"trace2": map[uint64][]*CallTreeP{
					0: []*CallTreeP{
						{"3", 0, 30, NoEndTime, NoEndTime, "trace2", "", "", []*CallTreeP{
							{"4", 0, 30, NoEndTime, NoEndTime, "trace2", "", "", nil},
						}},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			agg := NewBacktraceAggregatorP()
			for _, backtrace := range tt.backtraces {
				agg.Update(BacktraceP{
					ProfileID:   backtrace.profileID,
					ThreadID:    backtrace.threadID,
					TimestampNs: backtrace.timestampNs,
					Addresses:   backtrace.addresses,
				})
			}
			agg.Finalize()
			if diff := testutil.Diff(agg.ProfileIDToCallTreeInfo, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestPropagatesProfileID(t *testing.T) {
	traceID := "trace1"

	agg := NewBacktraceAggregatorP()
	agg.Update(BacktraceP{
		ProfileID: traceID,
		Addresses: []string{"2", "1", "0"},
	})
	agg.Finalize()
	gotCallTreeP := agg.ProfileIDToCallTreeInfo[traceID][0][0]
	if g, w := gotCallTreeP.ProfileID, traceID; g != w {
		t.Fatalf("trace ID mismatch: got %v want %v", g, w)
	}
}

func TestPropagatesThreadName(t *testing.T) {
	traceID := "trace1"
	threadID := uint64(1)
	threadName := "threadName"

	agg := NewBacktraceAggregatorP()
	agg.Update(BacktraceP{
		ProfileID:  traceID,
		ThreadID:   1,
		Addresses:  []string{"2", "1", "0"},
		ThreadName: threadName,
	})
	agg.Finalize()
	gotCallTreeP := agg.ProfileIDToCallTreeInfo[traceID][1][0]
	if g, w := gotCallTreeP.ThreadID, threadID; g != w {
		t.Fatalf("thread ID mismatch: got %v want %v", g, w)
	}
	if g, w := gotCallTreeP.ThreadName, threadName; g != w {
		t.Fatalf("thread name mismatch: got %v want %v", g, w)
	}
}

func TestPropagatesQueueName(t *testing.T) {
	traceID := "trace1"
	threadID := uint64(1)
	queueName := "queueName"

	agg := NewBacktraceAggregatorP()
	agg.Update(BacktraceP{
		ProfileID:  traceID,
		ThreadID:   1,
		Addresses:  []string{"2", "1", "0"},
		ThreadName: "threadName",
		QueueName:  queueName,
	})
	agg.Finalize()
	gotCallTreeP := agg.ProfileIDToCallTreeInfo[traceID][1][0]
	if g, w := gotCallTreeP.ThreadID, threadID; g != w {
		t.Fatalf("thread ID mismatch: got %v want %v", g, w)
	}
	if g, w := gotCallTreeP.ThreadName, queueName; g != w {
		t.Fatalf("queue name mismatch: got %v want %v", g, w)
	}
}

func TestPropagatesSessionKey(t *testing.T) {
	traceID := "trace1"
	sessionKey := "sessionKey"

	agg := NewBacktraceAggregatorP()
	agg.Update(BacktraceP{
		ProfileID:  traceID,
		Addresses:  []string{"2", "1", "0"},
		SessionKey: sessionKey,
	})
	agg.Finalize()
	if g, w := agg.ProfileIDToCallTreeInfo[traceID][0][0].SessionKey, sessionKey; g != w {
		t.Fatalf("session key mismatch on call tree: got %v want %v", g, w)
	}
}

func TestGetCallTreePThreadName(t *testing.T) {
	tests := []struct {
		name       string
		threadName string
		queueName  string
		want       string
	}{
		{
			name:       "thread name but no queue name",
			threadName: "threadName",
			want:       "threadName",
		},
		{
			name:      "queue name but no thread name",
			queueName: "queueName",
			want:      "queueName",
		},
		{
			name:       "queue name and thread name",
			queueName:  "queueName",
			threadName: "threadName",
			want:       "queueName",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := getCallTreeThreadName(tt.queueName, tt.threadName)
			if diff := testutil.Diff(got, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
