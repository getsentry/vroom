package snubautil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/getsentry/vroom/internal/testutil"
)

func TestFormatStatsMeta(t *testing.T) {
	tests := []struct {
		name        string
		dataset     string
		startStr    string
		endStr      string
		start       int64
		end         int64
		granularity int64
		err         string
	}{
		{
			name:        "format stats meta",
			dataset:     "profiles",
			startStr:    "2022-07-01T04:00:00.000000+00:00",
			endStr:      "2022-07-14T04:00:00.000000+00:00",
			start:       1656633600,
			end:         1657756800,
			granularity: 86400,
			err:         "",
		},
		{
			name:        "0 granularity",
			dataset:     "profiles",
			startStr:    "2022-07-01T04:00:00.000000+00:00",
			endStr:      "2022-07-14T04:00:00.000000+00:00",
			start:       0,
			end:         0,
			granularity: 0,
			err:         "invalid granularity: 0 must be greater than 0",
		},
		{
			name:        "bad start format",
			dataset:     "profiles",
			startStr:    "hi",
			endStr:      "2022-07-14:00:00.000000+00:00",
			start:       0,
			end:         0,
			granularity: 0,
			err:         `parsing time "hi" as "2006-01-02T15:04:05.000000+00:00": cannot parse "hi" as "2006"`,
		},
		{
			name:        "bad end format",
			dataset:     "profiles",
			startStr:    "2022-07-01T04:00:00.000000+00:00",
			endStr:      "hi",
			start:       0,
			end:         0,
			granularity: 0,
			err:         `parsing time "hi" as "2006-01-02T15:04:05.000000+00:00": cannot parse "hi" as "2006"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := FormatStatsMeta(tt.dataset, tt.startStr, tt.endStr, tt.granularity)

			if tt.err != "" {
				if err == nil {
					t.Fatal("expected error to be non-nil")
				} else if !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("expected error message to contain %q but was %q", tt.err, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected error to be nil %q", err.Error())
				}

				if meta.Dataset != tt.dataset {
					t.Fatalf(`expected dataset: "%v" but was "%v"`, tt.dataset, meta.Dataset)
				}

				if meta.Start != tt.start {
					t.Fatalf(`expected start: "%v" but was "%v"`, tt.start, meta.Start)
				}

				if meta.End != tt.end {
					t.Fatalf(`expected end: "%v" but was "%v"`, tt.end, meta.End)
				}

				if meta.Granularity != tt.granularity {
					t.Fatalf(`expected granularity: "%v" but was "%v"`, tt.granularity, meta.Granularity)
				}
			}
		})
	}
}

type SimpleRawStats struct {
	axes       []string
	data       map[string][]float64
	timestamps []int64
}

func (s SimpleRawStats) Axes() []string {
	return s.axes
}

func (s SimpleRawStats) TimestampAt(idx int) int64 {
	if idx >= len(s.timestamps) {
		return -1
	}
	return s.timestamps[idx]
}

func (s SimpleRawStats) ValueAt(axis string, idx int) (float64, error) {
	timeseries, ok := s.data[axis]
	if ok && idx < len(timeseries) {
		return timeseries[idx], nil
	}
	return 0, fmt.Errorf("no value for axis: %s", axis)
}

func TestFormatStatsBadAxis(t *testing.T) {
	_, _, err := FormatStats(
		SimpleRawStats{
			axes:       []string{"a"},
			data:       map[string][]float64{},
			timestamps: []int64{1656633600},
		},
		StatsMeta{
			Dataset:     "profiles",
			Start:       1656633600,
			End:         1657756800,
			Granularity: 86400,
		},
	)

	expectedErr := "no value for axis: a"
	if err == nil {
		t.Fatal("expected error to be non-nil")
	} else if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error message to contain %q but was %q", expectedErr, err.Error())
	}
}

func TestFormatStatsSimpleStats(t *testing.T) {
	timestamps, data, err := FormatStats(
		SimpleRawStats{
			axes: []string{"a", "b"},
			data: map[string][]float64{
				"a": []float64{0, 1, 2},
				"b": []float64{3, 4, 5},
			},
			timestamps: []int64{10, 25, 30},
		},
		StatsMeta{
			Dataset:     "profiles",
			Start:       10,
			End:         30,
			Granularity: 5,
		},
	)

	if err != nil {
		t.Fatalf("expected error to be nil but was %q", err.Error())
	}

	expectedTs := StatsTimestamps{10, 15, 20, 25, 30}
	if diff := testutil.Diff(timestamps, expectedTs); diff != "" {
		t.Fatalf(`expected timestamps "%v" but was "%v"`, timestamps, expectedTs)
	}

	nums := []float64{0, 1, 2, 3, 4, 5}
	expectedData := []StatsData{
		StatsData{
			Axis:   "a",
			Values: []*float64{&nums[0], nil, nil, &nums[1], &nums[2]},
		},
		StatsData{
			Axis:   "b",
			Values: []*float64{&nums[3], nil, nil, &nums[4], &nums[5]},
		},
	}

	if diff := testutil.Diff(data, expectedData); diff != "" {
		t.Fatalf(`expected timestamps "%v" but was "%v"`, data, expectedData)
	}
}
