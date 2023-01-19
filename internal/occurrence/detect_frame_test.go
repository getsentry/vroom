package occurrence

import (
	"testing"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/testutil"
)

func TestDetectFrameOnCallTree(t *testing.T) {
	tests := []struct {
		name string
		node *nodetree.Node
		want []*nodetree.Node
	}{
		{
			name: "Detect frame in call tree",
			node: &nodetree.Node{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				StartNS:       0,
				Children: []*nodetree.Node{
					&nodetree.Node{
						DurationNS:    5,
						EndNS:         5,
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child1-1",
						Package:       "package",
						Path:          "path",
						StartNS:       0,
						Children: []*nodetree.Node{
							&nodetree.Node{
								DurationNS:    5,
								EndNS:         5,
								Fingerprint:   0,
								IsApplication: true,
								Line:          0,
								Name:          "child2-1",
								Package:       "package",
								Path:          "path",
								StartNS:       0,
								Children: []*nodetree.Node{
									&nodetree.Node{
										DurationNS:    5,
										EndNS:         5,
										Fingerprint:   0,
										IsApplication: false,
										Line:          0,
										Name:          "CFReadStreamRead",
										Package:       "CoreFoundation",
										Path:          "path",
										StartNS:       0,
										Children:      []*nodetree.Node{},
									},
								},
							},
						},
					},
					&nodetree.Node{
						DurationNS:    5,
						EndNS:         10,
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child1-2",
						Package:       "package",
						Path:          "path",
						StartNS:       5,
						Children: []*nodetree.Node{
							&nodetree.Node{
								DurationNS:    5,
								EndNS:         10,
								Fingerprint:   0,
								IsApplication: true,
								Line:          0,
								Name:          "",
								Package:       "",
								Path:          "",
								StartNS:       5,
								Children: []*nodetree.Node{
									&nodetree.Node{
										DurationNS:    5,
										EndNS:         10,
										Fingerprint:   0,
										IsApplication: false,
										Line:          0,
										Name:          "child3-1",
										Package:       "package",
										Path:          "path",
										StartNS:       5,
										Children:      []*nodetree.Node{},
									},
								},
							},
						},
					},
				},
			},
			want: []*nodetree.Node{
				&nodetree.Node{
					DurationNS:    5,
					EndNS:         5,
					Fingerprint:   0,
					IsApplication: false,
					Line:          0,
					Name:          "CFReadStreamRead",
					Package:       "CoreFoundation",
					Path:          "path",
					StartNS:       0,
					Children:      []*nodetree.Node{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nodes []*nodetree.Node
			detectFrameInCallTree(tt.node, detectExactFrameMetadata[platform.Cocoa][0].FunctionsByPackage, &nodes)
			if diff := testutil.Diff(nodes, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
