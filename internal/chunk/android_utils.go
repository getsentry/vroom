package chunk

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/getsentry/vroom/internal/measurements"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/speedscope"
)

var member void

type void struct{}

func SpeedscopeFromAndroidChunks(chunks []AndroidChunk, startTS, endTS uint64) (speedscope.Output, error) {
	if len(chunks) == 0 {
		return speedscope.Output{}, nil
	}
	maxTsNS := uint64(0)
	threadSet := make(map[uint64]void)
	// fingerprint to method ID
	methodToID := make(map[uint32]uint64)
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].EndTimestamp() <= chunks[j].StartTimestamp()
	})

	mergedMeasurement := make(map[string]measurements.MeasurementV2)

	chunk := chunks[0]
	firstChunkStartTimestampNS := uint64(chunk.StartTimestamp() * 1e9)
	// Initially, adjustedChunkStartTimestampNS will just be the
	// chunk timestamp. If the chunk starts before the allowed
	// time range though, we only keep events that fall within
	// the range, set the startTimestamp to the start value of
	// the allowed range and adjust the relative ts of each
	// events.
	adjustedChunkStartTimestampNS := firstChunkStartTimestampNS
	buildTimestamp := chunk.Profile.TimestampGetter()
	// clean up the events in the first chunk
	events := make([]profile.AndroidEvent, 0, len(chunk.Profile.Events))
	methods := make([]profile.AndroidMethod, 0, len(chunk.Profile.Methods))
	// updates methods ID
	tmpMethodsID := make(map[uint64]uint64)
	for i, method := range chunk.Profile.Methods {
		id := uint64(i + 1)
		tmpMethodsID[method.ID] = id
		method.ID = id
		methodToID[method.Frame().Fingerprint()] = id
		methods = append(methods, method)
	}
	delta := int64(0)
	if firstChunkStartTimestampNS < startTS {
		delta = -int64(startTS - firstChunkStartTimestampNS)
		adjustedChunkStartTimestampNS = startTS
	}
	addTimeDelta := chunk.Profile.AddTimeDelta(delta)
	for _, event := range chunk.Profile.Events {
		ts := buildTimestamp(event.Time) + firstChunkStartTimestampNS
		if ts < startTS || ts > endTS {
			// we filter out events out of range
			continue
		}
		// If the event falls within allowed range, but the first chunk
		// begins before the start range (delta != 0), adjust the relative ts
		// of each event by subtracting the delta.
		if delta != 0 {
			err := addTimeDelta(&event)
			if err != nil {
				return speedscope.Output{}, err
			}
			// update ts
			ts = buildTimestamp(event.Time) + adjustedChunkStartTimestampNS
		}
		event.MethodID = tmpMethodsID[event.MethodID]
		events = append(events, event)
		maxTsNS = max(maxTsNS, ts)
	}
	for _, thread := range chunk.Profile.Threads {
		threadSet[thread.ID] = member
	}
	if len(chunk.Measurements) > 0 {
		err := json.Unmarshal(chunk.Measurements, &mergedMeasurement)
		if err != nil {
			return speedscope.Output{}, err
		}
	}

	// If chunk started before the allowed time range
	// update the chunk timestamp (firstChunkStartTimestampNS)
	// since later on, other chunks will use this to compute
	// the right offset (relative ts in nanoseconds).
	if delta != 0 {
		firstChunkStartTimestampNS = adjustedChunkStartTimestampNS
	}

	for i := 1; i < len(chunks); i++ {
		c := chunks[i]
		chunkStartTimestampNs := uint64(c.StartTimestamp() * 1e9)
		buildTimestamp := c.Profile.TimestampGetter()
		// Delta between the current chunk timestamp and the very first one.
		// This will be needed to correctly offset the events relative ts,
		// which need to be relative not to the start of this chunk, but to
		// the start of the very first one.
		delta := chunkStartTimestampNs - firstChunkStartTimestampNS
		addTimeDelta := c.Profile.AddTimeDelta(int64(delta))
		// updates methods ID
		tmpMethodsID = make(map[uint64]uint64)
		for _, method := range c.Profile.Methods {
			fingerprint := method.Frame().Fingerprint()
			if id, ok := methodToID[fingerprint]; !ok {
				newID := uint64(len(methodToID) + 1)
				methodToID[fingerprint] = newID
				tmpMethodsID[method.ID] = newID
				method.ID = newID
				methods = append(methods, method)
			} else {
				tmpMethodsID[method.ID] = id
			}
		}

		// filter events
		for _, event := range c.Profile.Events {
			ts := buildTimestamp(event.Time) + chunkStartTimestampNs
			if ts < startTS || ts > endTS {
				continue
			}
			event.MethodID = tmpMethodsID[event.MethodID]
			// Before adding the event, update its relative timestamp
			// which, in this case, should not be relative to the current
			// chunk timestamp, but rather relative to the very 1st one.
			err := addTimeDelta(&event)
			if err != nil {
				return speedscope.Output{}, err
			}
			ts = buildTimestamp(event.Time) + firstChunkStartTimestampNS
			events = append(events, event)
			maxTsNS = max(maxTsNS, ts)
		}
		// Update threads.
		for _, thread := range c.Profile.Threads {
			if _, ok := threadSet[thread.ID]; !ok {
				chunk.Profile.Threads = append(c.Profile.Threads, thread)
				threadSet[thread.ID] = member
			}
		}
		// In case we have measurements, merge them too.
		if len(c.Measurements) > 0 {
			var chunkMeasurements map[string]measurements.MeasurementV2
			err := json.Unmarshal(c.Measurements, &chunkMeasurements)
			if err != nil {
				return speedscope.Output{}, err
			}
			for k, measurement := range chunkMeasurements {
				if el, ok := mergedMeasurement[k]; ok {
					el.Values = append(el.Values, measurement.Values...)
					mergedMeasurement[k] = el
				} else {
					mergedMeasurement[k] = measurement
				}
			}
		}
	}
	chunk.Profile.Events = events
	chunk.Profile.Methods = methods
	chunk.DurationNS = maxTsNS - startTS

	s, err := chunk.Profile.Speedscope()
	if err != nil {
		return speedscope.Output{}, err
	}
	s.DurationNS = chunk.DurationNS
	s.Metadata.Timestamp = time.Unix(0, int64(firstChunkStartTimestampNS)).UTC()
	s.ChunkID = chunk.ID
	s.Platform = chunk.Platform

	if len(mergedMeasurement) > 0 {
		s.Measurements = mergedMeasurement
	}

	return s, nil
}
