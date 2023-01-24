package occurrence

import (
	"testing"
	"time"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/testutil"
)

func TestDetectFrameInCallTree(t *testing.T) {
	trueValue := true
	falseValue := false
	tests := []struct {
		name string
		node *nodetree.Node
		want map[nodeKey]nodeInfo
	}{
		{
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
				Children: []*nodetree.Node{
					&nodetree.Node{
						DurationNS:    uint64(20 * time.Millisecond),
						EndNS:         uint64(20 * time.Millisecond),
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child1-1",
						Package:       "package",
						Path:          "path",
						StartNS:       0,
						Children: []*nodetree.Node{
							&nodetree.Node{
								DurationNS:    uint64(20 * time.Millisecond),
								EndNS:         uint64(20 * time.Millisecond),
								Fingerprint:   0,
								IsApplication: true,
								Line:          0,
								Name:          "child2-1",
								Package:       "package",
								Path:          "path",
								StartNS:       0,
								Children: []*nodetree.Node{
									&nodetree.Node{
										DurationNS:    uint64(20 * time.Millisecond),
										EndNS:         uint64(20 * time.Millisecond),
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
			want: map[nodeKey]nodeInfo{
				nodeKey{
					Package:  "CoreFoundation",
					Function: "CFReadStreamRead",
				}: nodeInfo{
					Node: &nodetree.Node{
						DurationNS:    uint64(20 * time.Millisecond),
						EndNS:         uint64(20 * time.Millisecond),
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "CFReadStreamRead",
						Package:       "CoreFoundation",
						Path:          "path",
						StartNS:       0,
						Children:      []*nodetree.Node{},
					},
					StackTrace: []frame.Frame{
						frame.Frame{
							Function: "root",
							InApp:    &trueValue,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						frame.Frame{
							Function: "child1-1",
							InApp:    &falseValue,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						frame.Frame{
							Function: "child2-1",
							InApp:    &trueValue,
							Line:     0,
							Package:  "package",
							Path:     "path",
						},
						frame.Frame{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := make(map[nodeKey]nodeInfo)
			var stackTrace []frame.Frame
			detectFrameInCallTree(tt.node, detectFrameJobs[platform.Cocoa][0], nodes, &stackTrace)
			if diff := testutil.Diff(nodes, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
