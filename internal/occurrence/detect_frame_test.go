package occurrence

import (
	"testing"
	"time"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/testutil"
)

var (
	falseValue = false
	trueValue  = true
)

func TestDetectFrameInCallTree(t *testing.T) {
	tests := []struct {
		job  DetectExactFrameOptions
		name string
		node *nodetree.Node
		want map[nodeKey]nodeInfo
	}{
		{
			job: DetectExactFrameOptions{
				DurationThreshold: 16 * time.Millisecond,
				FunctionsByPackage: map[string]map[string]Category{
					"CoreFoundation": {
						"CFReadStreamRead": FileRead,
					},
				},
			},
			name: "Detect frame in call tree",
			node: &nodetree.Node{
				DurationNS:    uint64(30 * time.Millisecond),
				EndNS:         uint64(30 * time.Millisecond),
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				StartNS:       0,
				Frame: frame.Frame{
					Function: "root",
					InApp:    &trueValue,
					Package:  "package",
					Path:     "path",
				},
				Children: []*nodetree.Node{
					{
						DurationNS:    uint64(20 * time.Millisecond),
						EndNS:         uint64(20 * time.Millisecond),
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child1-1",
						Package:       "package",
						Path:          "path",
						StartNS:       0,
						Frame: frame.Frame{
							Function: "child1-1",
							InApp:    &falseValue,
							Package:  "package",
							Path:     "path",
						},
						Children: []*nodetree.Node{
							{
								DurationNS:    uint64(20 * time.Millisecond),
								EndNS:         uint64(20 * time.Millisecond),
								Fingerprint:   0,
								IsApplication: true,
								Line:          0,
								Name:          "child2-1",
								Package:       "package",
								Path:          "path",
								StartNS:       0,
								Frame: frame.Frame{
									Function: "child2-1",
									InApp:    &trueValue,
									Package:  "package",
									Path:     "path",
								},
								Children: []*nodetree.Node{
									{
										DurationNS:    uint64(20 * time.Millisecond),
										EndNS:         uint64(20 * time.Millisecond),
										Fingerprint:   0,
										IsApplication: false,
										Line:          0,
										Name:          "CFReadStreamRead",
										Package:       "CoreFoundation",
										Path:          "path",
										SampleCount:   4,
										StartNS:       0,
										Frame: frame.Frame{
											Function: "CFReadStreamRead",
											InApp:    &falseValue,
											Package:  "CoreFoundation",
											Path:     "path",
										},
										Children: []*nodetree.Node{},
									},
								},
							},
						},
					},
					{
						DurationNS:    5,
						EndNS:         10,
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child1-2",
						Package:       "package",
						Path:          "path",
						StartNS:       5,
						Frame: frame.Frame{
							Function: "child1-2",
							InApp:    &falseValue,
							Package:  "package",
							Path:     "path",
						},
						Children: []*nodetree.Node{
							{
								DurationNS:    5,
								EndNS:         10,
								Fingerprint:   0,
								IsApplication: true,
								Line:          0,
								Name:          "child2-1",
								Package:       "package",
								Path:          "path",
								StartNS:       5,
								Frame: frame.Frame{
									Function: "child2-1",
									InApp:    &trueValue,
									Package:  "package",
									Path:     "path",
								},
								Children: []*nodetree.Node{
									{
										DurationNS:    5,
										EndNS:         10,
										Fingerprint:   0,
										IsApplication: false,
										Line:          0,
										Name:          "child3-1",
										Package:       "package",
										Path:          "path",
										StartNS:       5,
										Frame: frame.Frame{
											Function: "child3-1",
											InApp:    &falseValue,
											Package:  "package",
											Path:     "path",
										},
										Children: []*nodetree.Node{},
									},
								},
							},
						},
					},
				},
			},
			want: map[nodeKey]nodeInfo{
				{
					Package:  "CoreFoundation",
					Function: "CFReadStreamRead",
				}: {
					Category: FileRead,
					Node: &nodetree.Node{
						DurationNS:    uint64(20 * time.Millisecond),
						EndNS:         uint64(20 * time.Millisecond),
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "CFReadStreamRead",
						Package:       "CoreFoundation",
						Path:          "path",
						SampleCount:   4,
						StartNS:       0,
						Frame: frame.Frame{
							Function: "CFReadStreamRead",
							InApp:    &falseValue,
							Package:  "CoreFoundation",
							Path:     "path",
						},
						Children: []*nodetree.Node{},
					},
					StackTrace: []frame.Frame{
						{
							Function: "root",
							InApp:    &trueValue,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "child1-1",
							InApp:    &falseValue,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "child2-1",
							InApp:    &trueValue,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "CFReadStreamRead",
							InApp:    &falseValue,
							Line:     0,
							Package:  "CoreFoundation",
							Path:     "path",
						},
					},
				},
			},
		},
		{
			job: DetectExactFrameOptions{
				DurationThreshold: 16 * time.Millisecond,
				FunctionsByPackage: map[string]map[string]Category{
					"CoreFoundation": {
						"CFReadStreamRead": FileRead,
					},
					"vroom": {
						"SuperShortFunction": FileRead,
					},
				},
			},
			name: "Do not detect frame in call tree under duration threshold",
			node: &nodetree.Node{
				DurationNS:    uint64(30 * time.Millisecond),
				EndNS:         uint64(30 * time.Millisecond),
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				StartNS:       0,
				Frame: frame.Frame{
					Function: "root",
					InApp:    &trueValue,
					Package:  "package",
					Path:     "path",
				},
				Children: []*nodetree.Node{
					{
						DurationNS:    uint64(20 * time.Millisecond),
						EndNS:         uint64(20 * time.Millisecond),
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child1-1",
						Package:       "package",
						Path:          "path",
						StartNS:       0,
						Frame: frame.Frame{
							Function: "child1-1",
							InApp:    &falseValue,
							Package:  "package",
							Path:     "path",
						},
						Children: []*nodetree.Node{
							{
								DurationNS:    uint64(20 * time.Millisecond),
								EndNS:         uint64(20 * time.Millisecond),
								Fingerprint:   0,
								IsApplication: true,
								Line:          0,
								Name:          "child2-1",
								Package:       "package",
								Path:          "path",
								StartNS:       0,
								Frame: frame.Frame{
									Function: "child2-1",
									InApp:    &trueValue,
									Package:  "package",
									Path:     "path",
								},
								Children: []*nodetree.Node{
									{
										DurationNS:    uint64(10 * time.Millisecond),
										EndNS:         uint64(10 * time.Millisecond),
										Fingerprint:   0,
										IsApplication: false,
										Line:          0,
										Name:          "SuperShortFunction",
										Package:       "vroom",
										Path:          "path",
										StartNS:       0,
										Frame: frame.Frame{
											Function: "SuperShortFunction",
											InApp:    &falseValue,
											Package:  "vroom",
											Path:     "path",
										},
										Children: []*nodetree.Node{},
									},
								},
							},
						},
					},
				},
			},
			want: map[nodeKey]nodeInfo{},
		},
		{
			job: DetectExactFrameOptions{
				DurationThreshold: 16 * time.Millisecond,
				SampleThreshold:   4,
				FunctionsByPackage: map[string]map[string]Category{
					"vroom": {
						"FunctionWithOneSample":   FileRead,
						"FunctionWithManySamples": FileRead,
					},
				},
			},
			name: "Do not detect frame in call tree under sample threshold",
			node: &nodetree.Node{
				DurationNS:    uint64(30 * time.Millisecond),
				EndNS:         uint64(30 * time.Millisecond),
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				StartNS:       0,
				Frame: frame.Frame{
					Function: "root",
					InApp:    &trueValue,
					Package:  "package",
					Path:     "path",
				},
				Children: []*nodetree.Node{
					{
						DurationNS:    uint64(20 * time.Millisecond),
						EndNS:         uint64(20 * time.Millisecond),
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child1-1",
						Package:       "package",
						Path:          "path",
						StartNS:       0,
						Frame: frame.Frame{
							Function: "child1-1",
							InApp:    &falseValue,
							Package:  "package",
							Path:     "path",
						},
						Children: []*nodetree.Node{
							{
								DurationNS:    uint64(20 * time.Millisecond),
								EndNS:         uint64(20 * time.Millisecond),
								Fingerprint:   0,
								IsApplication: true,
								Line:          0,
								Name:          "child2-1",
								Package:       "package",
								Path:          "path",
								StartNS:       0,
								Frame: frame.Frame{
									Function: "child2-1",
									InApp:    &trueValue,
									Package:  "package",
									Path:     "path",
								},
								Children: []*nodetree.Node{
									{
										DurationNS:    uint64(20 * time.Millisecond),
										EndNS:         uint64(20 * time.Millisecond),
										Fingerprint:   0,
										IsApplication: false,
										Line:          0,
										Name:          "FunctionWithOneSample",
										Package:       "vroom",
										Path:          "path",
										SampleCount:   1,
										StartNS:       0,
										Children:      []*nodetree.Node{},
										Frame: frame.Frame{
											Function: "FunctionWithOneSample",
											InApp:    &falseValue,
											Package:  "vroom",
											Path:     "path",
										},
									},
									{
										DurationNS:    uint64(20 * time.Millisecond),
										EndNS:         uint64(20 * time.Millisecond),
										Fingerprint:   0,
										IsApplication: true,
										Line:          0,
										Name:          "child3-1",
										Package:       "package",
										Path:          "path",
										StartNS:       0,
										Frame: frame.Frame{
											Function: "child3-1",
											InApp:    &trueValue,
											Package:  "package",
											Path:     "path",
										},
										Children: []*nodetree.Node{
											{
												DurationNS:    uint64(20 * time.Millisecond),
												EndNS:         uint64(20 * time.Millisecond),
												Fingerprint:   0,
												IsApplication: false,
												Line:          0,
												Name:          "FunctionWithManySamples",
												Package:       "vroom",
												Path:          "path",
												SampleCount:   4,
												StartNS:       0,
												Children:      []*nodetree.Node{},
												Frame: frame.Frame{
													Function: "FunctionWithManySamples",
													InApp:    &falseValue,
													Package:  "vroom",
													Path:     "path",
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
			want: map[nodeKey]nodeInfo{
				{
					Package:  "vroom",
					Function: "FunctionWithManySamples",
				}: {
					Category: FileRead,
					Node: &nodetree.Node{
						DurationNS:    uint64(20 * time.Millisecond),
						EndNS:         uint64(20 * time.Millisecond),
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "FunctionWithManySamples",
						Package:       "vroom",
						Path:          "path",
						SampleCount:   4,
						StartNS:       0,
						Children:      []*nodetree.Node{},
						Frame: frame.Frame{
							Function: "FunctionWithManySamples",
							InApp:    &falseValue,
							Package:  "vroom",
							Path:     "path",
						},
					},
					StackTrace: []frame.Frame{
						{
							Function: "root",
							InApp:    &trueValue,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "child1-1",
							InApp:    &falseValue,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "child2-1",
							InApp:    &trueValue,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "child3-1",
							InApp:    &trueValue,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "FunctionWithManySamples",
							InApp:    &falseValue,
							Line:     0,
							Package:  "vroom",
							Path:     "path",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := make(map[nodeKey]nodeInfo)
			var stackTrace []frame.Frame
			detectFrameInCallTree(tt.node, tt.job, nodes, &stackTrace)
			if diff := testutil.Diff(nodes, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
