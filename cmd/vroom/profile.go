package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/segmentio/kafka-go"
	"gocloud.dev/gcerrors"
	"google.golang.org/api/googleapi"

	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/occurrence"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/storageutil"
)

const (
	maxUniqueFunctionsPerProfile = 100
	unsampledProfileID           = "00000000000000000000000000000000"
)

func (env *environment) postProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)

	s := sentry.StartSpan(ctx, "processing")
	s.Description = "Read HTTP body"
	body, err := io.ReadAll(r.Body)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetContext("Profile metadata", map[string]interface{}{
		"Size": len(body),
	})

	var p profile.Profile
	s = sentry.StartSpan(ctx, "json.unmarshal")
	s.Description = "Unmarshal profile"
	err = json.Unmarshal(body, &p)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	orgID := p.OrganizationID()

	profilePlatform := p.Platform()

	hub.Scope().SetTags(map[string]string{
		"organization_id": strconv.FormatUint(orgID, 10),
		"platform":        string(profilePlatform),
		"profile_id":      p.ID(),
		"project_id":      strconv.FormatUint(p.ProjectID(), 10),
	})

	s = sentry.StartSpan(ctx, "processing")
	s.Description = "Normalize profile"
	p.Normalize()
	s.Finish()

	if !p.IsSampled() {
		// if we're dealing with an unsampled profile
		// we'll assign the special "000....00" profile ID
		// so that we can handle it accordingly either in
		// either of snuba/sentry/front-end
		p.SetProfileID(unsampledProfileID)
	}

	if p.IsSampled() {
		s = sentry.StartSpan(ctx, "gcs.write")
		s.Description = "Write profile to GCS"
		err = storageutil.CompressedWrite(ctx, env.storage, p.StoragePath(), p)
		s.Finish()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				// This is a transient error, we'll retry
				w.WriteHeader(http.StatusTooManyRequests)
			} else {
				// These errors won't be retried
				hub.CaptureException(err)
				if code := gcerrors.Code(err); code == gcerrors.FailedPrecondition {
					w.WriteHeader(http.StatusPreconditionFailed)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
			return
		}
	}

	s = sentry.StartSpan(ctx, "processing")
	s.Description = "Generate call trees"
	callTrees, err := p.CallTrees()
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(callTrees) > 0 {
		// if the profile was not sampled we skip find_occurrences since we're only
		// interested in extracting data to improve functions aggregations not in
		// using it for finding occurrences of an issue
		if p.IsSampled() {
			s = sentry.StartSpan(ctx, "processing")
			s.Description = "Find occurrences"
			occurrences := occurrence.Find(p, callTrees)
			s.Finish()

			// Filter in-place occurrences without a type.
			var i int
			for _, o := range occurrences {
				if o.Type != occurrence.NoneType {
					occurrences[i] = o
					i++
				}
			}
			occurrences = occurrences[:i]
			s = sentry.StartSpan(ctx, "processing")
			s.Description = "Build Kafka message batch"
			occurrenceMessages, err := occurrence.GenerateKafkaMessageBatch(occurrences)
			s.Finish()
			if err != nil {
				// Report the error but don't fail profile insertion
				hub.CaptureException(err)
			} else {
				s = sentry.StartSpan(ctx, "processing")
				s.Description = "Send occurrences to Kafka"
				err = env.occurrencesWriter.WriteMessages(ctx, occurrenceMessages...)
				s.Finish()
				if err != nil {
					// Report the error but don't fail profile insertion
					hub.CaptureException(err)
				}
			}
		}

		// Prepare call trees Kafka message
		s = sentry.StartSpan(ctx, "processing")
		s.Description = "Extract functions"
		functions := extractFunctionsFromCallTrees(callTrees)
		// Cap but don't filter out system frames.
		// Necessary until front end changes are in place.
		functionsDataset := capAndFilterFunctions(functions, false)
		s.Finish()

		s = sentry.StartSpan(ctx, "json.marshal")
		s.Description = "Marshal functions Kafka message"
		b, err := json.Marshal(buildFunctionsKafkaMessage(p, functionsDataset))
		s.Finish()
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		s = sentry.StartSpan(ctx, "processing")
		s.Description = "Send functions to Kafka"
		err = env.profilingWriter.WriteMessages(ctx, kafka.Message{
			Topic: env.config.CallTreesKafkaTopic,
			Value: b,
		})
		s.Finish()
		hub.Scope().SetContext("Call functions payload", map[string]interface{}{
			"Size": len(b),
		})
		if err != nil {
			hub.CaptureException(err)
		}
		if p.GetOptions().ProjectDSN != "" {
			s = sentry.StartSpan(ctx, "processing")
			s.Description = "Extract metrics from functions"
			// Cap and filter out system frames.
			functionsMetricPlatform := capAndFilterFunctions(functions, true)
			metrics, metricsSummary := extractMetricsFromFunctions(&p, functionsMetricPlatform)
			s.Finish()

			if len(metrics) > 0 {
				s = sentry.StartSpan(ctx, "processing")
				s.Description = "Send functions metrics to generic metrics platform"
				sendMetrics(ctx, &p, metrics)
				s.Finish()
			}

			// Only send a profile sample to the metrics_summary if it's an indexed profile
			if p.IsSampled() && len(metrics) > 0 {
				kafkaMessages, err := generateMetricSummariesKafkaMessageBatch(&p, metrics, metricsSummary)
				if err != nil {
					hub.CaptureException(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				err = env.metricSummaryWriter.WriteMessages(ctx, kafkaMessages...)
				if err != nil {
					hub.CaptureException(err)
				}
			}
		}
	}

	if p.IsSampled() {
		// Prepare profile Kafka message
		s = sentry.StartSpan(ctx, "processing")
		s.Description = "Marshal profile metadata Kafka message"
		b, err := json.Marshal(buildProfileKafkaMessage(p))
		s.Finish()
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		s = sentry.StartSpan(ctx, "processing")
		s.Description = "Send profile metadata to Kafka"
		err = env.profilingWriter.WriteMessages(ctx, kafka.Message{
			Topic: env.config.ProfilesKafkaTopic,
			Value: b,
		})
		s.Finish()
		hub.Scope().SetContext("Profile metadata Kafka payload", map[string]interface{}{
			"Size": len(b),
		})
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func extractFunctionsFromCallTrees(
	callTrees map[uint64][]*nodetree.Node,
) []nodetree.CallTreeFunction {
	functions := make(map[uint32]nodetree.CallTreeFunction, 0)

	for _, callTreesForThread := range callTrees {
		for _, callTree := range callTreesForThread {
			callTree.CollectFunctions(functions)
		}
	}

	functionsList := make([]nodetree.CallTreeFunction, 0, len(functions))
	for _, function := range functions {
		if function.SampleCount <= 1 {
			// if there's only ever a single sample for this function in
			// the profile, we skip over it to reduce the amount of data
			continue
		}
		functionsList = append(functionsList, function)
	}

	// sort the list in descending order, and take the top N results
	sort.SliceStable(functionsList, func(i, j int) bool {
		return functionsList[i].SumSelfTimeNS > functionsList[j].SumSelfTimeNS
	})

	return functionsList
}

func capAndFilterFunctions(functions []nodetree.CallTreeFunction, filterSystemFrames bool) []nodetree.CallTreeFunction {
	if !filterSystemFrames {
		if len(functions) > maxUniqueFunctionsPerProfile {
			return functions[:maxUniqueFunctionsPerProfile]
		}
		return functions
	}
	appFunctions := make([]nodetree.CallTreeFunction, 0, min(maxUniqueFunctionsPerProfile, len(functions)))
	for _, f := range functions {
		if !f.InApp {
			continue
		}
		appFunctions = append(appFunctions, f)
		if len(appFunctions) == maxUniqueFunctionsPerProfile {
			break
		}
	}
	return appFunctions
}

func (env *environment) getRawProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)
	ps := httprouter.ParamsFromContext(ctx)
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("organization_id", rawOrganizationID)

	rawProjectID := ps.ByName("project_id")
	projectID, err := strconv.ParseUint(rawProjectID, 10, 64)
	if err != nil {
		sentry.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("project_id", rawProjectID)

	profileID := ps.ByName("profile_id")
	_, err = uuid.Parse(profileID)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("profile_id", profileID)
	s := sentry.StartSpan(ctx, "profile.read")
	s.Description = "Read profile from GCS or Snuba"

	var p profile.Profile
	err = storageutil.UnmarshalCompressed(
		ctx,
		env.storage,
		profile.StoragePath(organizationID, projectID, profileID),
		&p,
	)
	s.Finish()
	if err != nil {
		if errors.Is(err, storageutil.ErrObjectNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var e *googleapi.Error
		if ok := errors.As(err, &e); ok {
			hub.Scope().SetContext("Google Cloud Storage Error", map[string]interface{}{
				"body":    e.Body,
				"code":    e.Code,
				"details": e.Details,
				"message": e.Message,
			})
		}
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()
	b, err := json.Marshal(p)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func (env *environment) getProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	qs := r.URL.Query()
	hub := sentry.GetHubFromContext(ctx)
	ps := httprouter.ParamsFromContext(ctx)
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("organization_id", rawOrganizationID)

	rawProjectID := ps.ByName("project_id")
	projectID, err := strconv.ParseUint(rawProjectID, 10, 64)
	if err != nil {
		sentry.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("project_id", rawProjectID)

	profileID := ps.ByName("profile_id")
	_, err = uuid.Parse(profileID)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("profile_id", profileID)
	s := sentry.StartSpan(ctx, "profile.read")
	s.Description = "Read profile from GCS or Snuba"

	var p profile.Profile
	err = storageutil.UnmarshalCompressed(
		ctx,
		env.storage,
		profile.StoragePath(organizationID, projectID, profileID),
		&p,
	)
	s.Finish()
	if err != nil {
		if errors.Is(err, storageutil.ErrObjectNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var e *googleapi.Error
		if ok := errors.As(err, &e); ok {
			hub.Scope().SetContext("Google Cloud Storage Error", map[string]interface{}{
				"body":    e.Body,
				"code":    e.Code,
				"details": e.Details,
				"message": e.Message,
			})
		}
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hub.Scope().SetTag("platform", string(p.Platform()))

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()

	var i interface{}

	if format := qs.Get("format"); format == "sample" && p.IsSampleFormat() {
		hub.Scope().SetTag("format", "sample")
		i = p
	} else {
		hub.Scope().SetTag("format", "speedscope")
		o, err := p.Speedscope()
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		i = o
	}

	b, err := json.Marshal(i)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
