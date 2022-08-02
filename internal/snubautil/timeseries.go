package snubautil

import (
	"encoding/json"
	"fmt"
	"time"
)

type (
	UnixTime time.Time

	StatsMeta struct {
		Dataset     string        `json:"dataset"`
		Start       UnixTime      `json:"start"`
		End         UnixTime      `json:"end"`
		Granularity time.Duration `json:"-"`
	}

	StatsData struct {
		Axis   string         `json:"axis"`
		Values []*interface{} `json:"values"`
	}
)

func (t UnixTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Unix())
}

func NewStatsMeta(dataset, rawStart, rawEnd string, rawGranularity uint64) (StatsMeta, error) {
	if rawGranularity <= 0 {
		return StatsMeta{}, fmt.Errorf("invalid granularity must be non zero: %d", rawGranularity)
	}
	granularity := time.Duration(rawGranularity) * time.Second

	start, err := truncateTime(rawStart, granularity)
	if err != nil {
		return StatsMeta{}, err
	}

	end, err := truncateTime(rawEnd, granularity)
	if err != nil {
		return StatsMeta{}, err
	}

	return StatsMeta{
		Dataset:     dataset,
		Start:       start,
		End:         end,
		Granularity: granularity,
	}, nil
}

func (m StatsMeta) Timestamps() []UnixTime {
	start := time.Time(m.Start)
	end := time.Time(m.End)
	n := end.Sub(start) / m.Granularity

	timestamps := make([]UnixTime, 0, n)

	for timestamp := start; !timestamp.After(end); timestamp = timestamp.Add(m.Granularity) {
		timestamps = append(timestamps, UnixTime(timestamp))
	}

	return timestamps
}

func FormatStats(meta StatsMeta, stats map[int64]map[string]interface{}, axes []string) ([]StatsData, []UnixTime) {
	timestamps := meta.Timestamps()

	dataMap := map[string]StatsData{}

	for _, axis := range axes {
		statsData := StatsData{
			Axis:   axis,
			Values: make([]*interface{}, 0, len(timestamps)),
		}

		for _, timestamp := range timestamps {
			bucket, ok := stats[time.Time(timestamp).Unix()]
			if !ok {
				statsData.Values = append(statsData.Values, nil)
			} else {
				value, ok := bucket[axis]
				if !ok {
					statsData.Values = append(statsData.Values, nil)
				} else {
					statsData.Values = append(statsData.Values, &value)
				}
			}
		}

		dataMap[axis] = statsData
	}

	data := make([]StatsData, 0, len(dataMap))
	for _, value := range dataMap {
		data = append(data, value)
	}

	return data, timestamps
}

func truncateTime(timeStr string, granularity time.Duration) (UnixTime, error) {
	rawTime, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		return UnixTime{}, err
	}
	return UnixTime(rawTime.Truncate(granularity)), nil
}
