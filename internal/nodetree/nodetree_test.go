package nodetree

import (
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/testutil"
)

const (
	fingerprintFoo  = 313808793
	fingerprintBar  = 45793645
	fingerprintBaz  = 3346457645
	fingerprintQux  = 4214270277
	fingerprintMain = 3605132115
)

func TestNodeTreeCollectFunctions(t *testing.T) {
	var minDepth uint
	tests := []struct {
		name     string
		platform platform.Platform
		node     Node
		want     map[uint32]CallTreeFunction
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
			want: map[uint32]CallTreeFunction{
				fingerprintFoo: {
					Fingerprint:   fingerprintFoo,
					InApp:         true,
					Function:      "foo",
					Package:       "foo",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
					MaxDuration:   10,
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
			want: map[uint32]CallTreeFunction{
				fingerprintFoo: {
					Fingerprint:   fingerprintFoo,
					InApp:         false,
					Function:      "foo",
					Package:       "foo",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
					MaxDuration:   10,
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
			want: map[uint32]CallTreeFunction{
				fingerprintFoo: {
					Fingerprint:   fingerprintFoo,
					InApp:         true,
					Function:      "foo",
					Package:       "foo",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
					MaxDuration:   10,
				},
				fingerprintBar: {
					Fingerprint:   fingerprintBar,
					InApp:         true,
					Function:      "bar",
					Package:       "bar",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
					MaxDuration:   10,
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
			want: map[uint32]CallTreeFunction{
				fingerprintFoo: {
					Fingerprint:   fingerprintFoo,
					InApp:         true,
					Function:      "foo",
					Package:       "foo",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
					MaxDuration:   10,
				},
				fingerprintBaz: {
					Fingerprint:   fingerprintBaz,
					InApp:         false,
					Function:      "baz",
					Package:       "baz",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
					MaxDuration:   10,
				},
			},
		},
		{
			name:     "multitple occurrences of same functions",
			platform: platform.Python,
			node: Node{
				DurationNS:    40,
				IsApplication: true,
				Frame: frame.Frame{
					Function: "main",
					Platform: "python",
					Package:  "main",
				},
				Children: []*Node{
					{
						DurationNS:    10,
						IsApplication: true,
						Frame: frame.Frame{
							Function: "foo",
							Package:  "foo",
							Platform: "python",
						},
						Children: []*Node{
							{
								DurationNS:    10,
								IsApplication: false,
								Frame: frame.Frame{
									Function: "bar",
									Package:  "bar",
									Platform: "python",
								},
								Children: []*Node{
									{
										DurationNS:    10,
										IsApplication: false,
										Frame: frame.Frame{
											Function: "baz",
											Package:  "baz",
											Platform: "python",
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
							Platform: "python",
						},
					},
					{
						DurationNS:    20,
						IsApplication: true,
						Frame: frame.Frame{
							Function: "foo",
							Package:  "foo",
							Platform: "python",
						},
						Children: []*Node{
							{
								DurationNS:    20,
								IsApplication: false,
								Frame: frame.Frame{
									Function: "bar",
									Package:  "bar",
									Platform: "python",
								},
								Children: []*Node{
									{
										DurationNS:    20,
										IsApplication: false,
										Frame: frame.Frame{
											Function: "baz",
											Package:  "baz",
											Platform: "python",
										},
									},
								},
							},
						},
					},
				},
			},
			want: map[uint32]CallTreeFunction{
				fingerprintFoo: {
					Fingerprint:   fingerprintFoo,
					InApp:         true,
					Function:      "foo",
					Package:       "foo",
					SelfTimesNS:   []uint64{10, 20},
					SumSelfTimeNS: 30,
					MaxDuration:   20,
				},
				fingerprintBaz: {
					Fingerprint:   fingerprintBaz,
					InApp:         false,
					Function:      "baz",
					Package:       "baz",
					SelfTimesNS:   []uint64{10, 20},
					SumSelfTimeNS: 30,
					MaxDuration:   20,
				},
				fingerprintQux: {
					Fingerprint:   fingerprintQux,
					InApp:         false,
					Function:      "qux",
					Package:       "qux",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
					MaxDuration:   10,
				},
				fingerprintMain: {
					Fingerprint:   fingerprintMain,
					InApp:         true,
					Function:      "main",
					Package:       "main",
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
					MaxDuration:   10,
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
					Platform: "android",
					Data: frame.Data{
						DeobfuscationStatus: "missing",
					},
				},
				Children: []*Node{
					{
						DurationNS:    10,
						IsApplication: true,
						Frame: frame.Frame{
							Function: "com.example.Thing.doStuff()",
							Package:  "com.example",
							Platform: "android",
							Data: frame.Data{
								DeobfuscationStatus: "deobfuscated",
							},
						},
					},
					{
						DurationNS:    10,
						IsApplication: true,
						Frame: frame.Frame{
							Data: frame.Data{
								DeobfuscationStatus: "partial",
							},
							Platform: "android",
							Function: "com.example.Thing.a()",
							Package:  "com.example",
						},
					},
				},
			},
			want: map[uint32]CallTreeFunction{
				261678695: {
					Fingerprint:   261678695,
					Function:      "com.example.Thing.doStuff()",
					Package:       "com.example",
					InApp:         true,
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
					MaxDuration:   10,
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
					Platform: "java",
				},
				Children: []*Node{
					{
						DurationNS:    10,
						IsApplication: true,
						Frame: frame.Frame{
							Function: "com.example.Thing.doStuff()",
							Package:  "com.example",
							Platform: "java",
						},
					},
				},
			},
			want: map[uint32]CallTreeFunction{
				261678695: {
					Fingerprint:   261678695,
					Function:      "com.example.Thing.doStuff()",
					Package:       "com.example",
					InApp:         true,
					SelfTimesNS:   []uint64{10},
					SumSelfTimeNS: 10,
					MaxDuration:   10,
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
					Platform: "cocoa",
				},
			},
			want: map[uint32]CallTreeFunction{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := make(map[uint32]CallTreeFunction)
			tt.node.CollectFunctions(results, "", 0, minDepth)
			if diff := testutil.Diff(results, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestIsSymbolicated(t *testing.T) {
	tests := []struct {
		name  string
		frame frame.Frame
		want  bool
	}{
		{
			name: "react-native-symbolicated",
			frame: frame.Frame{
				IsReactNative: true,
				Platform:      "javascript",
				Data:          frame.Data{JsSymbolicated: &testutil.True},
			},
			want: true,
		},
		{
			name: "react-native-not-symbolicated",
			frame: frame.Frame{
				IsReactNative: true,
				Platform:      "javascript",
				Data:          frame.Data{},
			},
			want: false,
		},
		{
			name: "browser-js",
			frame: frame.Frame{
				Platform: "javascript",
				Data:     frame.Data{},
			},
			want: true,
		},
		{
			name: "nodejs",
			frame: frame.Frame{
				Platform: "javascript",
				Data:     frame.Data{},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := testutil.Diff(isSymbolicatedFrame(tt.frame), tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
