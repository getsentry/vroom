package flamegraph

import (
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/sample"
)

var sampledProfile = sample.SampleProfile{
	Platform: "cocoa",
	Version:  "v1",
	Trace: sample.Trace{
		Frames: []frame.Frame{
			{
				Function: "a",
				Package:  "test.package",
				Path:     "/tmp",
			},
			{
				Function: "b",
				Package:  "test.package",
				Path:     "/tmp",
			},
			{
				Function: "c",
				Package:  "test.package",
				Path:     "/tmp",
			},
		}, //end frames
		Stacks: []sample.Stack{
			{1, 0}, // b,a
			{2},    // c
			{1, 0}, // b,a
			{0},    // a
		},
		Samples: []sample.Sample{
			{
				ElapsedSinceStartNS: 0,
				StackID:             0,
				ThreadID:            0,
			},
			{
				ElapsedSinceStartNS: 10,
				StackID:             1,
				ThreadID:            0,
			},
			{
				ElapsedSinceStartNS: 20,
				StackID:             2,
				ThreadID:            0,
			},
			{
				ElapsedSinceStartNS: 20,
				StackID:             3,
				ThreadID:            0,
			},
		}, // end Samples
	}, // end Trace
	Transactions: []sample.Transaction{
		{
			ActiveThreadID: 0,
		},
	}, // end Transactions
} // end prof definition
