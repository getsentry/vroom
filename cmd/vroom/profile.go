package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/segmentio/kafka-go"
	"gocloud.dev/gcerrors"
	"google.golang.org/api/googleapi"

	"github.com/getsentry/vroom/internal/metrics"
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
	defer r.Body.Close()

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

	orgID := p.OrganizationID()

	hub.Scope().SetContext("Profile metadata", map[string]interface{}{
		"organization_id": strconv.FormatUint(orgID, 10),
		"profile_id":      p.ID(),
		"project_id":      strconv.FormatUint(p.ProjectID(), 10),
		"size":            len(body),
	})

	profilePlatform := p.Platform()

	hub.Scope().SetTags(map[string]string{
		"platform": string(profilePlatform),
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
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				// These are transient errors, we'll retry.
				w.WriteHeader(http.StatusTooManyRequests)
			} else {
				if code := gcerrors.Code(err); code == gcerrors.FailedPrecondition {
					// This indicates a duplicate, we won't retry.
					w.WriteHeader(http.StatusPreconditionFailed)
				} else {
					if hub != nil {
						hub.CaptureException(err)
					}
					// These errors won't be retried.
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
		functions := metrics.ExtractFunctionsFromCallTrees(callTrees)
		// Cap but don't filter out system frames.
		// Necessary until front end changes are in place.
		functionsDataset := metrics.CapAndFilterFunctions(functions, maxUniqueFunctionsPerProfile, false)
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
