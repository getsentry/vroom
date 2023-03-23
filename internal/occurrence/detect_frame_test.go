package occurrence

import (
	"testing"
	"time"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/testutil"
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
					InApp:    &testutil.True,
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
							InApp:    &testutil.False,
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
									InApp:    &testutil.True,
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
											InApp:    &testutil.False,
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
							InApp:    &testutil.False,
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
									InApp:    &testutil.True,
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
											InApp:    &testutil.False,
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
					Node: nodetree.Node{
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
							InApp:    &testutil.False,
							Package:  "CoreFoundation",
							Path:     "path",
						},
					},
					StackTrace: []frame.Frame{
						{
							Function: "root",
							InApp:    &testutil.True,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "child1-1",
							InApp:    &testutil.False,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "child2-1",
							InApp:    &testutil.True,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "CFReadStreamRead",
							InApp:    &testutil.False,
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
					InApp:    &testutil.True,
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
							InApp:    &testutil.False,
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
									InApp:    &testutil.True,
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
											InApp:    &testutil.False,
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
					InApp:    &testutil.True,
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
							InApp:    &testutil.False,
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
									InApp:    &testutil.True,
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
											InApp:    &testutil.False,
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
											InApp:    &testutil.True,
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
													InApp:    &testutil.False,
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
					Node: nodetree.Node{
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
						Frame: frame.Frame{
							Function: "FunctionWithManySamples",
							InApp:    &testutil.False,
							Package:  "vroom",
							Path:     "path",
						},
					},
					StackTrace: []frame.Frame{
						{
							Function: "root",
							InApp:    &testutil.True,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "child1-1",
							InApp:    &testutil.False,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "child2-1",
							InApp:    &testutil.True,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "child3-1",
							InApp:    &testutil.True,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "FunctionWithManySamples",
							InApp:    &testutil.False,
							Line:     0,
							Package:  "vroom",
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
						"AnotherLeafFunction": FileRead,
						"LeafFunction":        FileRead,
						"RandomFunction":      FileRead,
					},
				},
			},
			name: "Detect deeper frame in call tree",
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
					InApp:    &testutil.True,
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
						Name:          "AnotherLeafFunction",
						Package:       "CoreFoundation",
						Path:          "path",
						StartNS:       0,
						Frame: frame.Frame{
							Function: "AnotherLeafFunction",
							InApp:    &testutil.False,
							Package:  "CoreFoundation",
							Path:     "path",
						},
						Children: []*nodetree.Node{},
					},
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
							InApp:    &testutil.False,
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
								Name:          "RandomFunction",
								Package:       "CoreFoundation",
								Path:          "path",
								StartNS:       0,
								Frame: frame.Frame{
									Function: "RandomFunction",
									InApp:    &testutil.True,
									Package:  "CoreFoundation",
									Path:     "path",
								},
								Children: []*nodetree.Node{
									{
										DurationNS:    uint64(20 * time.Millisecond),
										EndNS:         uint64(20 * time.Millisecond),
										Fingerprint:   0,
										IsApplication: false,
										Line:          0,
										Name:          "LeafFunction",
										Package:       "CoreFoundation",
										Path:          "path",
										StartNS:       0,
										Frame: frame.Frame{
											Function: "LeafFunction",
											InApp:    &testutil.False,
											Package:  "CoreFoundation",
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
					Function: "AnotherLeafFunction",
				}: {
					Category: FileRead,
					Node: nodetree.Node{
						DurationNS:    uint64(20 * time.Millisecond),
						EndNS:         uint64(20 * time.Millisecond),
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "AnotherLeafFunction",
						Package:       "CoreFoundation",
						Path:          "path",
						StartNS:       0,
						Frame: frame.Frame{
							Function: "AnotherLeafFunction",
							InApp:    &testutil.False,
							Package:  "CoreFoundation",
							Path:     "path",
						},
					},
					StackTrace: []frame.Frame{
						{
							Function: "root",
							InApp:    &testutil.True,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "AnotherLeafFunction",
							InApp:    &testutil.False,
							Line:     0,
							Package:  "CoreFoundation",
							Path:     "path",
						},
					},
				},
				{
					Package:  "CoreFoundation",
					Function: "LeafFunction",
				}: {
					Category: FileRead,
					Node: nodetree.Node{
						DurationNS:    uint64(20 * time.Millisecond),
						EndNS:         uint64(20 * time.Millisecond),
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "LeafFunction",
						Package:       "CoreFoundation",
						Path:          "path",
						StartNS:       0,
						Frame: frame.Frame{
							Function: "LeafFunction",
							InApp:    &testutil.False,
							Package:  "CoreFoundation",
							Path:     "path",
						},
					},
					StackTrace: []frame.Frame{
						{
							Function: "root",
							InApp:    &testutil.True,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "child1-1",
							InApp:    &testutil.False,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						{
							Function: "RandomFunction",
							InApp:    &testutil.True,
							Line:     0,
							Package:  "CoreFoundation",
							Path:     "path",
						},
						{
							Function: "LeafFunction",
							InApp:    &testutil.False,
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
						"RandomFunction": FileRead,
					},
				},
			},
			name: "Detect first frame",
			node: &nodetree.Node{
				DurationNS:    uint64(30 * time.Millisecond),
				EndNS:         uint64(30 * time.Millisecond),
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "RandomFunction",
				Package:       "CoreFoundation",
				Path:          "path",
				StartNS:       0,
				Frame: frame.Frame{
					Function: "RandomFunction",
					InApp:    &testutil.True,
					Package:  "CoreFoundation",
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
							InApp:    &testutil.False,
							Package:  "package",
							Path:     "path",
						},
					},
					{
						DurationNS:    uint64(20 * time.Millisecond),
						EndNS:         uint64(20 * time.Millisecond),
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child1-2",
						Package:       "package",
						Path:          "path",
						StartNS:       0,
						Frame: frame.Frame{
							Function: "child1-2",
							InApp:    &testutil.False,
							Package:  "package",
							Path:     "path",
						},
					},
				},
			},
			want: map[nodeKey]nodeInfo{
				{
					Package:  "CoreFoundation",
					Function: "RandomFunction",
				}: {
					Category: FileRead,
					Node: nodetree.Node{
						DurationNS:    uint64(30 * time.Millisecond),
						EndNS:         uint64(30 * time.Millisecond),
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "RandomFunction",
						Package:       "CoreFoundation",
						Path:          "path",
						StartNS:       0,
						Frame: frame.Frame{
							Function: "RandomFunction",
							InApp:    &testutil.True,
							Package:  "CoreFoundation",
							Path:     "path",
						},
					},
					StackTrace: []frame.Frame{
						{
							Function: "RandomFunction",
							InApp:    &testutil.True,
							Line:     0,
							Package:  "CoreFoundation",
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
			detectFrameInCallTree(tt.node, tt.job, nodes)
			if diff := testutil.Diff(nodes, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
