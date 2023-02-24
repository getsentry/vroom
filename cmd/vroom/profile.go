package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/occurrence"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/storageutil"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/googleapi"
)

func (env *environment) postProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)

	var p profile.Profile

	s := sentry.StartSpan(ctx, "json.unmarshal")
	s.Description = "Unmarshal Snuba profile"
	err := json.NewDecoder(r.Body).Decode(&p)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	orgID := p.OrganizationID()

	hub.Scope().SetTags(map[string]string{
		"organization_id": strconv.FormatUint(orgID, 10),
		"platform":        string(p.Platform()),
		"profile_id":      p.ID(),
		"project_id":      strconv.FormatUint(p.ProjectID(), 10),
	})

	s = sentry.StartSpan(ctx, "processing")
	s.Description = "Normalize profile"
	p.Normalize()
	s.Finish()

	s = sentry.StartSpan(ctx, "processing")
	s.Description = "Generate call trees"
	callTrees, err := p.CallTrees()
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s = sentry.StartSpan(ctx, "gcs.write")
	s.Description = "Write profile to GCS"
	err = storageutil.CompressedWrite(ctx, env.profilesBucket, p.StoragePath(), p)
	s.Finish()
	if err != nil {
		var e *googleapi.Error
		if ok := errors.As(err, &e); ok {
			w.WriteHeader(http.StatusBadGateway)
		} else if errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	messages := make([]kafka.Message, 0, 2)

	if len(callTrees) > 0 {
		s = sentry.StartSpan(ctx, "processing")
		s.Description = "Find occurrences"
		occurrences := occurrence.Find(p, callTrees)
		s.Finish()

		if env.occurrencesInserter != nil {
			// Log occurrences with a link to access to corresponding profiles
			// It will be removed when the issue platform UI is functional
			s = sentry.StartSpan(ctx, "bq.write")
			s.Description = "Write occurrences to BigQuery"
			err := env.occurrencesInserter.Put(ctx, occurrences)
			s.Finish()
			if err != nil {
				hub.CaptureException(err)
			}
		}

		if _, enabled := env.config.OccurrencesEnabledOrganizations[orgID]; enabled {
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
			messages, err := occurrence.GenerateKafkaMessageBatch(occurrences)
			s.Finish()
			if err != nil {
				// Report the error but don't fail profile insertion
				hub.CaptureException(err)
			} else {
				s = sentry.StartSpan(ctx, "processing")
				s.Description = "Send occurrences to Kafka"
				err = env.occurrencesWriter.WriteMessages(ctx, messages...)
				s.Finish()
				if err != nil {
					// Report the error but don't fail profile insertion
					hub.CaptureException(err)
				}
			}
		}

		// Prepare call trees Kafka message
		s = sentry.StartSpan(ctx, "processing")
		s.Description = "Collapse call trees"
		for threadID, callTreesForThread := range callTrees {
			collapsedCallTrees := make([]*nodetree.Node, 0, len(callTreesForThread))
			for _, callTree := range callTreesForThread {
				collapsedCallTrees = append(collapsedCallTrees, callTree.Collapse()...)
			}
			callTrees[threadID] = collapsedCallTrees
		}
		s.Finish()

		s = sentry.StartSpan(ctx, "json.marshal")
		s.Description = "Marshal call trees Kafka message"
		b, err := json.Marshal(buildCallTreesKafkaMessage(p, callTrees))
		s.Finish()
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		messages = append(messages, kafka.Message{
			Topic: env.config.CallTreesKafkaTopic,
			Value: b,
		})
	}

	// Prepare profile Kafka message
	s = sentry.StartSpan(ctx, "processing")
	s.Description = "Marshal profile Kafka message"
	b, err := json.Marshal(buildProfileKafkaMessage(p))
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	messages = append(messages, kafka.Message{
		Topic: env.config.ProfilesKafkaTopic,
		Value: b,
	})

	// Send all messages to Kafka
	s = sentry.StartSpan(ctx, "processing")
	s.Description = "Send messages to Kafka"
	err = env.profilingWriter.WriteMessages(ctx, messages...)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
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
	err = storageutil.UnmarshalCompressed(ctx, env.profilesBucket, profile.StoragePath(organizationID, projectID, profileID), &p)
	s.Finish()
	if err != nil {
		var e *googleapi.Error
		if ok := errors.As(err, &e); ok {
			if e.Code == 404 {
				w.WriteHeader(http.StatusNotFound)
				return
			}
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
	err = storageutil.UnmarshalCompressed(ctx, env.profilesBucket, profile.StoragePath(organizationID, projectID, profileID), &p)
	s.Finish()
	if err != nil {
		var e *googleapi.Error
		if ok := errors.As(err, &e); ok {
			if e.Code == 404 {
				w.WriteHeader(http.StatusNotFound)
				return
			}
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

	o, err := p.Speedscope()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	b, err := json.Marshal(o)
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
