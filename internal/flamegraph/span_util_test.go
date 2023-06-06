package flamegraph

import (
	"testing"
	"time"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/testutil"
)

func TestMergeIntervals(t *testing.T) {
	inputIntervals := []SpanInterval{
		{Start: 8, End: 11},
		{Start: 3, End: 6},
		{Start: 7, End: 12},
		{Start: 1, End: 3},
	}

	expectedResult := []SpanInterval{
		{Start: 1, End: 6},
		{Start: 7, End: 12},
	}

	result := mergeIntervals(&inputIntervals)

	if diff := testutil.Diff(result, expectedResult); diff != "" {
		t.Fatalf("Result mismatch: got - want +\n%s", diff)
	}
}

func TestGetTotalOvelappingDuration(t *testing.T) {
	tests := []struct {
		name      string
		node      nodetree.Node
		intervals []SpanInterval
		output    uint64
	}{
		{
			/*
				|------------------------- NODE -------------------------|

				|---- SPAN 1 ----|	|---- SPAN 2 ----|
			*/
			name: "node overlaps both spans",
			node: nodetree.Node{
				StartNS: 0,
				EndNS:   uint64(60 * time.Millisecond),
			},
			intervals: []SpanInterval{
				{Start: 0, End: uint64(20 * time.Millisecond)},
				{Start: uint64(20 * time.Millisecond), End: uint64(40 * time.Millisecond)},
			},
			output: uint64(40 * time.Millisecond),
		},
		{
			/*
						|------------------------- NODE -------------------------|

				|---- SPAN 1 ----|										|---- SPAN 2 ----|
			*/
			name: "node partially overlaps both spans",
			node: nodetree.Node{
				StartNS: uint64(30 * time.Millisecond),
				EndNS:   uint64(90 * time.Millisecond),
			},
			intervals: []SpanInterval{
				{Start: uint64(20 * time.Millisecond), End: uint64(40 * time.Millisecond)},
				{Start: uint64(80 * time.Millisecond), End: uint64(100 * time.Millisecond)},
			},
			output: uint64(20 * time.Millisecond),
		},
		{
			/*
				|------------------------- NODE -------------------------|

					|---- SPAN 1 ----|										|---- SPAN 2 ----|
			*/
			name: "node overlaps only one span",
			node: nodetree.Node{
				StartNS: 0,
				EndNS:   uint64(80 * time.Millisecond),
			},
			intervals: []SpanInterval{
				{Start: uint64(20 * time.Millisecond), End: uint64(40 * time.Millisecond)},
				{Start: uint64(90 * time.Millisecond), End: uint64(100 * time.Millisecond)},
			},
			output: uint64(20 * time.Millisecond),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			intervals := mergeIntervals(&test.intervals)
			result := getTotalOverlappingDuration(&test.node, &intervals)

			if diff := testutil.Diff(result, test.output); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestSliceCallTree(t *testing.T) {
	tests := []struct {
		name      string
		callTree  []*nodetree.Node
		intervals []SpanInterval
		output    []*nodetree.Node
	}{
		{
			/*
				|------------------------- NODE -------------------------|
					        |							|
					|---- CHILD 1 ----|			|---- CHILD 2 ----|
				|------------------------ SPAN 1 ------------------------|
			*/
			name: "call tree and span exact overlap",
			callTree: []*nodetree.Node{
				{
					StartNS:     0,
					EndNS:       uint64(100 * time.Millisecond),
					SampleCount: 10,
					Children: []*nodetree.Node{
						{StartNS: uint64(10 * time.Millisecond), EndNS: uint64(30 * time.Millisecond), SampleCount: 2},
						{StartNS: uint64(60 * time.Millisecond), EndNS: uint64(80 * time.Millisecond), SampleCount: 2},
					},
				},
			},
			intervals: []SpanInterval{
				{Start: 0, End: uint64(100 * time.Millisecond)},
			},
			output: []*nodetree.Node{
				{
					StartNS:     0,
					EndNS:       uint64(100 * time.Millisecond),
					SampleCount: 10,
					Children: []*nodetree.Node{
						{StartNS: uint64(10 * time.Millisecond), EndNS: uint64(30 * time.Millisecond), SampleCount: 2},
						{StartNS: uint64(60 * time.Millisecond), EndNS: uint64(80 * time.Millisecond), SampleCount: 2},
					},
				},
			},
		},
		{
			/*
				|------------------------- NODE -------------------------|
					        |
					|---- CHILD 1 ----|
					|--------- SPAN 1 ---------|
			*/
			name: "call tree and span exact overlap",
			callTree: []*nodetree.Node{
				{
					StartNS:     0,
					EndNS:       uint64(100 * time.Millisecond),
					SampleCount: 10,
					Children: []*nodetree.Node{
						{StartNS: uint64(10 * time.Millisecond), EndNS: uint64(50 * time.Millisecond), SampleCount: 4},
					},
				},
			},
			intervals: []SpanInterval{
				{Start: uint64(10 * time.Millisecond), End: uint64(60 * time.Millisecond)},
			},
			output: []*nodetree.Node{
				{
					StartNS:     0,
					EndNS:       uint64(100 * time.Millisecond),
					SampleCount: 5,
					Children: []*nodetree.Node{
						{StartNS: uint64(10 * time.Millisecond), EndNS: uint64(50 * time.Millisecond), SampleCount: 4},
					},
				},
			},
		},
		{
			/*
				|----------------------------- NODE -----------------------------|
				 	     |					    |					   |
				 |---- CHILD 1 ----|    |---- CHILD 1 ----|    |---- CHILD 1 ----|
													  		   |----- SPAN 1 ----|
			*/
			name: "span overlaps only one child and part of the parent call frame",
			callTree: []*nodetree.Node{
				{
					StartNS:     0,
					EndNS:       uint64(100 * time.Millisecond),
					SampleCount: 10,
					Children: []*nodetree.Node{
						{StartNS: uint64(10 * time.Millisecond), EndNS: uint64(30 * time.Millisecond), SampleCount: 2},
						{StartNS: uint64(30 * time.Millisecond), EndNS: uint64(50 * time.Millisecond), SampleCount: 2},
						{StartNS: uint64(80 * time.Millisecond), EndNS: uint64(100 * time.Millisecond), SampleCount: 2},
					},
				},
			},
			intervals: []SpanInterval{
				{Start: uint64(80 * time.Millisecond), End: uint64(100 * time.Millisecond)},
			},
			output: []*nodetree.Node{
				{
					StartNS:     0,
					EndNS:       uint64(100 * time.Millisecond),
					SampleCount: 2,
					Children: []*nodetree.Node{
						{StartNS: uint64(80 * time.Millisecond), EndNS: uint64(100 * time.Millisecond), SampleCount: 2},
					},
				},
			},
		},
		{
			/*
				|------------------------- NODE -------------------------|
					        		|
					|------------ CHILD 1 ------------|
					|--- SPAN 1 ---|   |--- SPAN 1 ---|
			*/
			name: "multiple spans overlap",
			callTree: []*nodetree.Node{
				{
					StartNS:     0,
					EndNS:       uint64(100 * time.Millisecond),
					SampleCount: 10,
					Children: []*nodetree.Node{
						{StartNS: uint64(10 * time.Millisecond), EndNS: uint64(60 * time.Millisecond), SampleCount: 5},
					},
				},
			},
			intervals: []SpanInterval{
				{Start: uint64(10 * time.Millisecond), End: uint64(30 * time.Millisecond)},
				{Start: uint64(40 * time.Millisecond), End: uint64(60 * time.Millisecond)},
			},
			output: []*nodetree.Node{
				{
					StartNS:     0,
					EndNS:       uint64(100 * time.Millisecond),
					SampleCount: 4,
					Children: []*nodetree.Node{
						{StartNS: uint64(10 * time.Millisecond), EndNS: uint64(60 * time.Millisecond), SampleCount: 4},
					},
				},
			},
		},
		{ // this simulate the scenario where the sampling frequency
			// could not be respected (Python native code holding the GIL,
			// php, etc.)
			/*
				|---------------- NODE ----------------|

						 |------ SPAN 1 ------|
			*/
			name: "defective sampling",
			callTree: []*nodetree.Node{
				{
					StartNS:     0,
					EndNS:       uint64(1000 * time.Millisecond),
					SampleCount: 2,
				},
			},
			intervals: []SpanInterval{
				{Start: uint64(250 * time.Millisecond), End: uint64(750 * time.Millisecond)},
			},
			output: []*nodetree.Node{
				{
					StartNS:     0,
					EndNS:       uint64(1000 * time.Millisecond),
					SampleCount: 2,
				},
			},
		},
	} // end test list

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			intervals := mergeIntervals(&test.intervals)
			result := sliceCallTree(&test.callTree, &intervals)
			if diff := testutil.Diff(result, test.output); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
