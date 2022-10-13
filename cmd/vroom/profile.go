package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"cloud.google.com/go/storage"
	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/snubautil"
	"github.com/getsentry/vroom/internal/storageutil"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/googleapi"
)

type PostProfileResponse struct {
	CallTrees map[uint64][]*nodetree.Node `json:"call_trees"`
}

func (env *environment) postProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)

	var p profile.Profile

	s := sentry.StartSpan(ctx, "json.unmarshal")
	s.Description = "Unmarshal Snuba profile"
	err := json.NewDecoder(r.Body).Decode(&p)
	s.Finish()
	if err != nil {
		log.Err(err).Msg("profile can't be unmarshaled")
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	hub.Scope().SetTags(map[string]string{
		"organization_id": strconv.FormatUint(p.OrganizationID(), 10),
		"project_id":      strconv.FormatUint(p.ProjectID(), 10),
		"profile_id":      p.ID(),
	})

	s = sentry.StartSpan(ctx, "calltree")
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

	s = sentry.StartSpan(ctx, "json.marshal")
	s.Description = "Marshal call trees"
	defer s.Finish()

	b, err := json.Marshal(PostProfileResponse{
		CallTrees: callTrees,
	})
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func getRawProfile(ctx context.Context, organizationID, projectID uint64, profileID string, profilesBucket *storage.BucketHandle, snuba snubautil.Client) (profile.Profile, error) {
	var p profile.Profile
	err := storageutil.UnmarshalCompressed(ctx, profilesBucket, profile.StoragePath(organizationID, projectID, profileID), &p)
	if err != nil {
		if !errors.Is(err, storage.ErrObjectNotExist) {
			return profile.Profile{}, err
		}
		sqb, err := snuba.NewQuery(ctx, "profiles")
		if err != nil {
			return profile.Profile{}, err
		}
		sp, err := snubautil.GetProfile(organizationID, projectID, profileID, sqb)
		if err != nil {
			return profile.Profile{}, err
		}
		p = sp
	}

	return p, nil
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

	p, err := getRawProfile(ctx, organizationID, projectID, profileID, env.profilesBucket, env.snuba)
	s.Finish()
	if err != nil {
		if errors.Is(err, snubautil.ErrProfileNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
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

	p, err := getRawProfile(ctx, organizationID, projectID, profileID, env.profilesBucket, env.snuba)
	s.Finish()
	if err != nil {
		if errors.Is(err, snubautil.ErrProfileNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	hub.Scope().SetTag("platform", p.Platform())

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()

	var b []byte
	switch p.Platform() {
	case "typescript":
		b = p.Raw()
	default:
		o, err := p.Speedscope()
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		b, err = json.Marshal(o)
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
