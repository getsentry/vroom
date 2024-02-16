package main

import (
	"testing"

	"github.com/getsentry/vroom/internal/testutil"
)

func TestGetFlamegraphNumWorkers(t *testing.T) {
	const minNumWorkers = 5
	tests := []struct {
		name          string
		numProfiles   int
		minNumWorkers int
		output        int
	}{
		{
			name:          "less profiles than minNumWorkers",
			numProfiles:   4,
			minNumWorkers: minNumWorkers,
			output:        4,
		},
		{
			name:          "as many profiles profiles as minNumWorkers",
			numProfiles:   5,
			minNumWorkers: minNumWorkers,
			output:        5,
		},
		{
			name:          "100s profiles",
			numProfiles:   100,
			minNumWorkers: minNumWorkers,
			output:        5,
		},
		{
			name:          "101s profiles",
			numProfiles:   101,
			minNumWorkers: minNumWorkers,
			output:        6,
		},
		{
			name:          "130s profiles",
			numProfiles:   130,
			minNumWorkers: minNumWorkers,
			output:        7,
		},
		{
			name:          "200s profiles",
			numProfiles:   200,
			minNumWorkers: minNumWorkers,
			output:        10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if diff := testutil.Diff(getFlamegraphNumWorkers(test.numProfiles, test.minNumWorkers), test.output); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
