package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/occurrence"
)

func (env *environment) postRegressed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	produceToKafka := r.URL.Query().Get("produce") != "false"

	regressedFunctions, err := decodeRegressedFunctionPayload(ctx, r)
	if err != nil {
		sentry.CaptureException(ctx, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	emitted := []occurrence.RegressedFunction{}
	occurrences := []*occurrence.Occurrence{}
	for _, regressedFunction := range regressedFunctions {
		s := sentry.StartSpan(ctx, "processing")
		s.Description = "Generating occurrence for payload"
		occurrence, err := occurrence.ProcessRegressedFunction(ctx, env.storage, regressedFunction, readJobs)
		s.Finish()
		if err != nil {
			sentry.CaptureException(ctx, err)
			continue
		} else if occurrence == nil {
			continue
		}
		emitted = append(emitted, regressedFunction)
		occurrences = append(occurrences, occurrence)
	}

	s := sentry.StartSpan(ctx, "json.marshal")
	data := struct {
		Occurrences        int                            `json:"occurrences"`
		Emitted            []occurrence.RegressedFunction `json:"emitted"`
		OccurrencePayloads *[]*occurrence.Occurrence      `json:"occurrence_payloads,omitempty"`
	}{Occurrences: len(occurrences), Emitted: emitted}
	if !produceToKafka {
		data.OccurrencePayloads = &occurrences
	}
	b, err := json.Marshal(data)
	s.Finish()
	if err != nil {
		sentry.CaptureException(ctx, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if produceToKafka {
		occurrenceMessages, err := occurrence.GenerateKafkaMessageBatch(occurrences)
		if err != nil {
			sentry.CaptureException(ctx, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		s = sentry.StartSpan(ctx, "processing")
		s.Description = "Send occurrences to Kafka"
		err = env.occurrencesWriter.WriteMessages(ctx, occurrenceMessages...)
		s.Finish()
		if err != nil {
			sentry.CaptureException(ctx, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func decodeRegressedFunctionPayload(ctx context.Context, r *http.Request) ([]occurrence.RegressedFunction, error) {
	s := sentry.StartSpan(ctx, "processing")
	s.Description = "Decoding payload"
	defer s.Finish()

	var regressedFunctions []occurrence.RegressedFunction
	err := json.NewDecoder(r.Body).Decode(&regressedFunctions)
	return regressedFunctions, err
}
