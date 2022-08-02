package snubautil

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/vroom/internal/testutil"
)

func TestNewStatsMeta(t *testing.T) {
	tests := []struct {
		name        string
		dataset     string
		startStr    string
		endStr      string
		start       time.Time
		end         time.Time
		granularity uint64
		err         string
	}{
		{
			name:        "format stats meta",
			dataset:     "profiles",
			startStr:    "2022-07-01T04:00:00.000000+00:00",
			endStr:      "2022-07-14T04:00:00.000000+00:00",
			start:       time.Date(2022, time.July, 1, 0, 0, 0, 0, time.UTC),
			end:         time.Date(2022, time.July, 14, 0, 0, 0, 0, time.UTC),
			granularity: 86400,
			err:         "",
		},
		{
			name:        "0 granularity",
			dataset:     "profiles",
			startStr:    "2022-07-01T04:00:00.000000+00:00",
			endStr:      "2022-07-14T04:00:00.000000+00:00",
			start:       time.Time{},
			end:         time.Time{},
			granularity: 0,
			err:         "invalid granularity must be non zero: 0",
		},
		{
			name:        "bad start format",
			dataset:     "profiles",
			startStr:    "hi",
			endStr:      "2022-07-14:00:00.000000+00:00",
			start:       time.Time{},
			end:         time.Time{},
			granularity: 86400,
			err:         `cannot parse "hi"`,
		},
		{
			name:        "bad end format",
			dataset:     "profiles",
			startStr:    "2022-07-01T04:00:00.000000+00:00",
			endStr:      "hi",
			start:       time.Time{},
			end:         time.Time{},
			granularity: 86400,
			err:         `cannot parse "hi"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := NewStatsMeta(tt.dataset, tt.startStr, tt.endStr, tt.granularity)

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

				if !time.Time(meta.Start).Equal(tt.start) {
					t.Fatalf(`expected start: "%v" but was "%v"`, tt.start, time.Time(meta.Start))
				}

				if !time.Time(meta.End).Equal(tt.end) {
					t.Fatalf(`expected end: "%v" but was "%v"`, tt.end, time.Time(meta.End))
				}

				if meta.Granularity != time.Duration(tt.granularity)*time.Second {
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

func TestFormatStatsSimpleStats(t *testing.T) {
	granularity := 5 * time.Second
	data, timestamps := FormatStats(
		StatsMeta{
			Dataset:     "profiles",
			Start:       UnixTime(time.Unix(10, 0)),
			End:         UnixTime(time.Unix(30, 0)),
			Granularity: granularity,
		},
		map[int64]map[string]interface{}{
			10: map[string]interface{}{
				"a": 0,
				"b": 3,
			},
			25: map[string]interface{}{
				"a": 1,
				"b": 4,
			},
			30: map[string]interface{}{
				"a": 2,
				"b": 5,
			},
		},
		[]string{"a", "b"},
	)

	expectedTs := make([]time.Time, 0, 5)
	actualTs := make([]time.Time, 0, 5)
	for i := 0; i < 5; i += 1 {
		expectedTs = append(expectedTs, time.Unix(10, 0).Add(time.Duration(i)*granularity))
		actualTs = append(actualTs, time.Time(timestamps[i]))
	}
	if diff := testutil.Diff(actualTs, expectedTs); diff != "" {
		t.Fatalf(`expected timestamps "%v" but was "%v"`, expectedTs, actualTs)
	}

	nums := []interface{}{0, 1, 2, 3, 4, 5}
	expectedData := []StatsData{
		StatsData{
			Axis:   "a",
			Values: []*interface{}{&nums[0], nil, nil, &nums[1], &nums[2]},
		},
		StatsData{
			Axis:   "b",
			Values: []*interface{}{&nums[3], nil, nil, &nums[4], &nums[5]},
		},
	}

	if diff := testutil.Diff(data, expectedData); diff != "" {
		t.Fatalf(`expected data "%v" but was "%v"`, expectedData, data)
	}
}
