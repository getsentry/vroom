package nodetree

import (
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/testutil"
)

const (
	fingerprintFoo  = 7819648313903568793
	fingerprintBar  = 7981615219744620909
	fingerprintBaz  = 14780661156850099245
	fingerprintQux  = 14955844843120965
	fingerprintMain = 6027741833354933075
)

func TestNodeTreeCollectFunctions(t *testing.T) {
	tests := []struct {
		name     string
		platform platform.Platform
		node     Node
		want     map[uint64]CallTreeFunction
	}{
		{
			name:     "single application node",
			platform: platform.Python,
			node: Node{
				DurationNS:    10,
				IsApplication: true,
				Frame: frame.Frame{
					Function: "foo",
					Package:  "foo",
				},
			},
			want: map[uint64]CallTreeFunction{
				fingerprintFoo: {
					Fingerprint:   fingerprintFoo,
					InApp:         true,
					Function:      "foo",
					Package:       "foo",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
				},
			},
		},
		{
			name:     "single system node",
			platform: platform.Python,
			node: Node{
				DurationNS:    10,
				IsApplication: false,
				Frame: frame.Frame{
					Function: "foo",
					Package:  "foo",
				},
			},
			want: map[uint64]CallTreeFunction{
				fingerprintFoo: {
					Fingerprint:   fingerprintFoo,
					InApp:         false,
					Function:      "foo",
					Package:       "foo",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
				},
			},
		},
		{
			name:     "non leaf node with non zero self time",
			platform: platform.Python,
			node: Node{
				DurationNS:    20,
				IsApplication: true,
				Frame: frame.Frame{
					Function: "foo",
					Package:  "foo",
				},
				Children: []*Node{
					{
						DurationNS:    10,
						IsApplication: true,
						Frame: frame.Frame{
							Function: "bar",
							Package:  "bar",
						},
					},
				},
			},
			want: map[uint64]CallTreeFunction{
				fingerprintFoo: {
					Fingerprint:   fingerprintFoo,
					InApp:         true,
					Function:      "foo",
					Package:       "foo",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
				},
				fingerprintBar: {
					Fingerprint:   fingerprintBar,
					InApp:         true,
					Function:      "bar",
					Package:       "bar",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
				},
			},
		},
		{
			name:     "application node wrapping system nodes of same duration",
			platform: platform.Python,
			node: Node{
				DurationNS:    10,
				IsApplication: true,
				Frame: frame.Frame{
					Function: "main",
					Package:  "main",
				},
				Children: []*Node{
					{
						DurationNS:    10,
						IsApplication: true,
						Frame: frame.Frame{
							Function: "foo",
							Package:  "foo",
						},
						Children: []*Node{
							{
								DurationNS:    10,
								IsApplication: false,
								Frame: frame.Frame{
									Function: "bar",
									Package:  "bar",
								},
								Children: []*Node{
									{
										DurationNS:    10,
										IsApplication: false,
										Frame: frame.Frame{
											Function: "baz",
											Package:  "baz",
										},
									},
								},
							},
						},
					},
				},
			},
			want: map[uint64]CallTreeFunction{
				fingerprintFoo: {
					Fingerprint:   fingerprintFoo,
					InApp:         true,
					Function:      "foo",
					Package:       "foo",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
				},
				fingerprintBaz: {
					Fingerprint:   fingerprintBaz,
					InApp:         false,
					Function:      "baz",
					Package:       "baz",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
				},
			},
		},
		{
			name:     "mutitple occurrences of same functions",
			platform: platform.Python,
			node: Node{
				DurationNS:    40,
				IsApplication: true,
				Frame: frame.Frame{
					Function: "main",
					Package:  "main",
				},
				Children: []*Node{
					{
						DurationNS:    10,
						IsApplication: true,
						Frame: frame.Frame{
							Function: "foo",
							Package:  "foo",
						},
						Children: []*Node{
							{
								DurationNS:    10,
								IsApplication: false,
								Frame: frame.Frame{
									Function: "bar",
									Package:  "bar",
								},
								Children: []*Node{
									{
										DurationNS:    10,
										IsApplication: false,
										Frame: frame.Frame{
											Function: "baz",
											Package:  "baz",
										},
									},
								},
							},
						},
					},
					{
						DurationNS:    10,
						IsApplication: false,
						Frame: frame.Frame{
							Function: "qux",
							Package:  "qux",
						},
					},
					{
						DurationNS:    20,
						IsApplication: true,
						Frame: frame.Frame{
							Function: "foo",
							Package:  "foo",
						},
						Children: []*Node{
							{
								DurationNS:    20,
								IsApplication: false,
								Frame: frame.Frame{
									Function: "bar",
									Package:  "bar",
								},
								Children: []*Node{
									{
										DurationNS:    20,
										IsApplication: false,
										Frame: frame.Frame{
											Function: "baz",
											Package:  "baz",
										},
									},
								},
							},
						},
					},
				},
			},
			want: map[uint64]CallTreeFunction{
				fingerprintFoo: {
					Fingerprint:   fingerprintFoo,
					InApp:         true,
					Function:      "foo",
					Package:       "foo",
					SelfTimesNS:   []uint64{10, 20},
					SumSelfTimeNS: 30,
				},
				fingerprintBaz: {
					Fingerprint:   fingerprintBaz,
					InApp:         false,
					Function:      "baz",
					Package:       "baz",
					SelfTimesNS:   []uint64{10, 20},
					SumSelfTimeNS: 30,
				},
				fingerprintQux: {
					Fingerprint:   fingerprintQux,
					InApp:         false,
					Function:      "qux",
					Package:       "qux",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
				},
				fingerprintMain: {
					Fingerprint:   fingerprintMain,
					InApp:         true,
					Function:      "main",
					Package:       "main",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
				},
			},
		},
		{
			name:     "obfuscated android frames",
			platform: platform.Android,
			node: Node{
				DurationNS:    20,
				IsApplication: true,
				Frame: frame.Frame{
					Function: "a.B()",
					Package:  "a",
				},
				Children: []*Node{
					{
						DurationNS:    10,
						IsApplication: true,
						Frame: frame.Frame{
							Function: "com.example.Thing.doStuff()",
							Package:  "com.example",
						},
					},
					{
						DurationNS:    10,
						IsApplication: true,
						Frame: frame.Frame{
							Data: frame.Data{
								DeobfuscationStatus: "missing",
							},
							Function: "com.example.Thing.doStuff()",
							Package:  "com.example",
						},
					},
					{
						DurationNS:    10,
						IsApplication: true,
						Frame: frame.Frame{
							Data: frame.Data{
								DeobfuscationStatus: "partial",
							},
							Function: "com.example.Thing.doStuff()",
							Package:  "com.example",
						},
					},
				},
			},
			want: map[uint64]CallTreeFunction{
				414680583044130407: {
					Fingerprint:   414680583044130407,
					Function:      "com.example.Thing.doStuff()",
					Package:       "com.example",
					InApp:         true,
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
				},
			},
		},
		{
			name:     "obfuscated java frames",
			platform: platform.Java,
			node: Node{
				DurationNS:    20,
				IsApplication: true,
				Frame: frame.Frame{
					Function: "a.B()",
					Package:  "a",
				},
				Children: []*Node{
					{
						DurationNS:    10,
						IsApplication: true,
						Frame: frame.Frame{
							Function: "com.example.Thing.doStuff()",
							Package:  "com.example",
						},
					},
				},
			},
			want: map[uint64]CallTreeFunction{
				414680583044130407: {
					Fingerprint:   414680583044130407,
					Function:      "com.example.Thing.doStuff()",
					Package:       "com.example",
					InApp:         true,
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
				},
			},
		},
		{
			name:     "cocoa main frame",
			platform: platform.Cocoa,
			node: Node{
				DurationNS:    10,
				IsApplication: true,
				Frame: frame.Frame{
					Function: "main",
					Package:  "iOS-Swift",
				},
			},
			want: map[uint64]CallTreeFunction{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := make(map[uint64]CallTreeFunction)
			tt.node.CollectFunctions(tt.platform, results)
			if diff := testutil.Diff(results, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
