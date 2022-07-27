package snubautil

import (
	"fmt"
	"time"
)

type (
	StatsMeta struct {
		Dataset     string `json:"dataset"`
		Start       int64  `json:"start"`
		End         int64  `json:"end"`
		Granularity int64  `json:"-"`
	}

	StatsTimestamps []int64

	StatsData struct {
		Axis   string     `json:"axis"`
		Values []*float64 `json:"values"`
	}

	RawStats interface {
		Axes() []string
		TimestampAt(idx int) int64
		ValueAt(axis string, idx int) (float64, error)
	}
)

const TIME_LAYOUT = "2006-01-02T15:04:05.000000+00:00"

func FormatStatsMeta(datasetStr, startStr, endStr string, granularity int64) (StatsMeta, error) {
	start, err := time.Parse(TIME_LAYOUT, startStr)
	if err != nil {
		return StatsMeta{}, err
	}

	end, err := time.Parse(TIME_LAYOUT, endStr)
	if err != nil {
		return StatsMeta{}, err
	}

	if granularity <= 0 {
		return StatsMeta{}, fmt.Errorf("invalid granularity: %d must be greater than 0", granularity)
	}

	return StatsMeta{
		Dataset:     datasetStr,
		Start:       start.Unix() / granularity * granularity,
		End:         end.Unix() / granularity * granularity,
		Granularity: granularity,
	}, nil
}

func FormatStats(rawStats RawStats, meta StatsMeta) (StatsTimestamps, []StatsData, error) {
	n := (meta.End-meta.Start)/meta.Granularity + 1
	timestamps := make([]int64, n, n)

	data := make(map[string]StatsData)
	for _, axis := range rawStats.Axes() {
		data[axis] = StatsData{
			Values: make([]*float64, n, n),
			Axis:   axis,
		}
	}

	rawIdx := 0

	for i, timestamp := 0, meta.Start; timestamp <= meta.End; i, timestamp = i+1, timestamp+meta.Granularity {
		timestamps[i] = timestamp

		if rawStats.TimestampAt(rawIdx) == timestamp {
			for _, axis := range rawStats.Axes() {
				value, err := rawStats.ValueAt(axis, rawIdx)
				if err != nil {
					return []int64{}, []StatsData{}, err
				}
				data[axis].Values[i] = &value
			}

			rawIdx += 1
		}
	}

	statsData := make([]StatsData, 0, len(data))
	for _, axisData := range data {
		statsData = append(statsData, axisData)
	}

	return timestamps, statsData, nil
}
