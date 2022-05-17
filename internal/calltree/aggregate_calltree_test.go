package calltree

import (
	"testing"

	"github.com/getsentry/vroom/internal/testutil"
)

func TestCallTreeAggregation(t *testing.T) {
	fs := func() []float64 {
		return nil
	}

	fsv := func(v float64) []float64 {
		return []float64{v}
	}

	fsm := func(fsuantiles ...[]float64) []float64 {
		var v []float64
		for _, fs := range fsuantiles {
			v = append(v, fs...)
		}
		return v
	}

	tests := []struct {
		name         string
		callTrees    []*AggregateCallTree
		targetImage  string
		targetSymbol string
		want         map[string]*AggregateCallTree
	}{
		{
			name: "single root call tree",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fs(), fs(), nil},
				}},
			},
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fs(), fs(), nil},
				}},
			},
		},
		{
			name: "multiple unique root call tree with different libraries but same function names",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fs(), fs(), nil},
				}},
				{"lib3", "func1", "func1-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib4", "func2", "func2-demangled", 0, "", fs(), fs(), nil},
				}},
			},
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fs(), fs(), nil},
				}},
				"dbe1669ed9e48c9b626ef69ea59a3e53": {"lib3", "func1", "func1-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib4", "func2", "func2-demangled", 0, "", fs(), fs(), nil},
				}},
			},
		},
		{
			name: "multiple unique root call tree with different function names but same library names",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fs(), fs(), nil},
				}},
				{"lib1", "func3", "func3-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func4", "func4-demangled", 0, "", fs(), fs(), nil},
				}},
			},
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fs(), fs(), nil},
				}},
				"8970a630a332f173f66bb7582c3db245": {"lib1", "func3", "func3-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func4", "func4-demangled", 0, "", fs(), fs(), nil},
				}},
			},
		},
		{
			name: "multiple unique root call tree with different function names and different library names",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fs(), fs(), nil},
				}},
				{"lib3", "func3", "func3-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib4", "func4", "func4-demangled", 0, "", fs(), fs(), nil},
				}},
			},
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fs(), fs(), nil},
				}},
				"47dac447d6f79f61b178adccea348678": {"lib3", "func3", "func3-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib4", "func4", "func4-demangled", 0, "", fs(), fs(), nil},
				}},
			},
		},
		{
			name: "merges same call tree",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
				{"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(800), fsv(2100), nil},
				}},
			},
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fsm(fsv(100), fsv(500)), fsm(fsv(1200), fsv(1800)), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsm(fsv(200), fsv(800)), fsm(fsv(1500), fsv(2100)), nil},
				}},
			},
		},
		{
			name: "does not merge partial call trees before target symbol",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
				{"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), nil},
			},
			targetImage:  "lib2",
			targetSymbol: "func2",
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
			},
		},
		{
			name: "does not merge partial call trees before target symbol, update order reversed",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), nil},
				{"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
			},
			targetImage:  "lib2",
			targetSymbol: "func2",
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
			},
		},
		{
			name: "does not merge partial call trees when target symbol is specified but doesn't exist",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
				{"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), nil},
			},
			targetImage:  "lib3",
			targetSymbol: "func3",
			want:         map[string]*AggregateCallTree{},
		},
		{
			name: "merges two of three partial call trees",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
				{"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), nil},
				{"lib1", "func1", "func1-demangled", 0, "", fsv(2000), fsv(3000), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(400), fsv(3000), []*AggregateCallTree{
						{"lib3", "func3", "func3-demangled", 0, "", fsv(0), fsv(20), nil},
					}},
				}},
			},
			targetImage:  "lib2",
			targetSymbol: "func2",
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fsm(fsv(100), fsv(2000)), fsm(fsv(1200), fsv(3000)), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsm(fsv(200), fsv(400)), fsm(fsv(1500), fsv(3000)), []*AggregateCallTree{
						{"lib3", "func3", "func3-demangled", 0, "", fsv(0), fsv(20), nil},
					}},
				}},
			},
		},
		{
			name: "does not merge partial call trees when there is no target symbol",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
				{"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), nil},
			},
			targetSymbol: "",
			targetImage:  "",
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
				"6a38e2e870f603ddb3cbad0244dc49a5": {"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), nil},
			},
		},
		{
			name: "does not merge partial call trees when tree different before target symbol",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
				{"lib3", "func3", "func3-demangled", 0, "", fsv(500), fsv(1800), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(100), fsv(1500), nil},
				}},
			},
			targetImage:  "lib2",
			targetSymbol: "func2",
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
				"1a129cabae4222a3c041c32534cffd70": {"lib3", "func3", "func3-demangled", 0, "", fsv(500), fsv(1800), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(100), fsv(1500), nil},
				}},
			},
		},
		{
			name: "merges partial call trees when not linear",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
					{"lib4", "func4", "func4-demangled", 0, "", fs(), fs(), nil},
				}},
				{"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(100), fsv(1500), nil},
					{"lib5", "func5", "func5-demangled", 0, "", fs(), fs(), nil},
				}},
			},
			targetImage:  "lib2",
			targetSymbol: "func2",
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fsm(fsv(100), fsv(500)), fsm(fsv(1200), fsv(1800)), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsm(fsv(200), fsv(100)), fsm(fsv(1500), fsv(1500)), nil},
				}},
			},
		},
		{
			name: "merges partial call trees that are different after target symbol",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), []*AggregateCallTree{
						{"lib3", "func3", "func3-demangled", 0, "", fs(), fs(), nil},
					}},
				}},
				{"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(800), fsv(2100), []*AggregateCallTree{
						{"lib4", "func4", "func4-demangled", 0, "", fs(), fs(), nil},
					}},
				}},
			},
			targetImage:  "lib2",
			targetSymbol: "func2",
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fsm(fsv(100), fsv(500)), fsm(fsv(1200), fsv(1800)), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsm(fsv(200), fsv(800)), fsm(fsv(1500), fsv(2100)), []*AggregateCallTree{
						{"lib3", "func3", "func3-demangled", 0, "", fs(), fs(), nil},
						{"lib4", "func4", "func4-demangled", 0, "", fs(), fs(), nil},
					}},
				}},
			},
		},
		{
			name: "does not merge same call tree with same root but different leaf",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
				{"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), []*AggregateCallTree{
					{"lib3", "func3", "func3-demangled", 0, "", fsv(800), fsv(2100), nil},
				}},
			},
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
				"227f7c481ee415323fff91517e846ab0": {"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), []*AggregateCallTree{
					{"lib3", "func3", "func3-demangled", 0, "", fsv(800), fsv(2100), nil},
				}},
			},
		},
		{
			name: "does not merge by comparing demangled symbol",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "SAME", 0, "", fsv(200), fsv(1500), nil},
				}},
				{"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), []*AggregateCallTree{
					{"lib3", "func3", "SAME", 0, "", fsv(800), fsv(2100), nil},
				}},
			},
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"lib2", "func2", "SAME", 0, "", fsv(200), fsv(1500), nil},
				}},
				"227f7c481ee415323fff91517e846ab0": {"lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), []*AggregateCallTree{
					{"lib3", "func3", "SAME", 0, "", fsv(800), fsv(2100), nil},
				}},
			},
		},
		{
			name: "uses demangled symbol from second call tree if not provided",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "", 0, "", fs(), fs(), nil},
				}},
				{"lib1", "func1", "func1-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fs(), fs(), nil},
				}},
			},
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 0, "", fs(), fs(), nil},
				}},
			},
		},
		{
			name: "biases towards newer value for path and line",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "func1-demangled", 1, "path1", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 2, "path2", fs(), fs(), nil},
				}},
				{"lib1", "func1", "func1-demangled", 3, "path3", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 4, "path4", fs(), fs(), nil},
				}},
			},
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "func1-demangled", 3, "path3", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "func2-demangled", 4, "path4", fs(), fs(), nil},
				}},
			},
		},
		{
			name: "merge same call tree",
			callTrees: []*AggregateCallTree{
				{"42851161-C416-463C-853E-E7C310D25FC0/lib1", "func1", "func1-demangled", 0, "", fsv(100), fsv(1200), []*AggregateCallTree{
					{"74646356-49C5-4D9E-B0C8-9DD5E62F9768/lib2", "func2", "func2-demangled", 0, "", fsv(200), fsv(1500), nil},
				}},
				{"F5F7FB85-A61A-4D44-9926-8FFA42768364/lib1", "func1", "func1-demangled", 0, "", fsv(500), fsv(1800), []*AggregateCallTree{
					{"F84148F0-883D-4AD0-BC3C-3D4EEAA9D3F4/lib2", "func2", "func2-demangled", 0, "", fsv(800), fsv(2100), nil},
				}},
			},
			want: map[string]*AggregateCallTree{
				"d39e48044f2412b016d894e74e8695d7": {"42851161-C416-463C-853E-E7C310D25FC0/lib1", "func1", "func1-demangled", 0, "", fsm(fsv(100), fsv(500)), fsm(fsv(1200), fsv(1800)), []*AggregateCallTree{
					{"74646356-49C5-4D9E-B0C8-9DD5E62F9768/lib2", "func2", "func2-demangled", 0, "", fsm(fsv(200), fsv(800)), fsm(fsv(1500), fsv(2100)), nil},
				}},
			},
		},
		{
			name: "do not duplicate calltrees if target method is recursive",
			callTrees: []*AggregateCallTree{
				{"lib1", "func1", "", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "", 0, "", fs(), fs(), []*AggregateCallTree{
						{"lib2", "func2", "", 0, "", fs(), fs(), []*AggregateCallTree{
							{"lib2", "func2", "", 0, "", fs(), fs(), []*AggregateCallTree{
								{"lib3", "func3", "", 0, "", fsv(800), fsv(2100), nil},
							}},
						}},
					}},
				}},
				{"lib1", "func1", "", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "", 0, "", fs(), fs(), nil},
				}},
			},
			want: map[string]*AggregateCallTree{
				"445ede0fa4873a99350f5119a0259e80": {"lib1", "func1", "", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "", 0, "", fs(), fs(), []*AggregateCallTree{
						{"lib2", "func2", "", 0, "", fs(), fs(), []*AggregateCallTree{
							{"lib2", "func2", "", 0, "", fs(), fs(), []*AggregateCallTree{
								{"lib3", "func3", "", 0, "", fsv(800), fsv(2100), nil},
							}},
						}},
					}},
				}},
				"d39e48044f2412b016d894e74e8695d7": {"lib1", "func1", "", 0, "", fs(), fs(), []*AggregateCallTree{
					{"lib2", "func2", "", 0, "", fs(), fs(), nil},
				}},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			agg := NewCallTreeAggregator()
			for _, tree := range tt.callTrees {
				if _, err := agg.Update(tree, tt.targetImage, tt.targetSymbol); err != nil {
					t.Fatal(err)
				}
			}
			if diff := testutil.Diff(agg.UniqueRootCallTrees, tt.want); diff != "" {
				t.Fatalf("result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
