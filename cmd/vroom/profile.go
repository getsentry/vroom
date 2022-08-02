package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/aggregate"
	"github.com/getsentry/vroom/internal/android"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/snubautil"
	"github.com/getsentry/vroom/internal/storageutil"
	"github.com/rs/zerolog/log"
)

type PostProfileResponse struct {
	CallTrees map[uint64][]*nodetree.Node `json:"call_trees"`
}

func (env *environment) postProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)

	s := sentry.StartSpan(ctx, "request.body")
	s.Description = "Read request body"
	body, err := io.ReadAll(r.Body)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s = sentry.StartSpan(ctx, "json.unmarshal")
	s.Description = "Unmarshal Snuba profile"
	var profile snubautil.Profile
	err = json.Unmarshal(body, &profile)
	s.Finish()
	if err != nil {
		log.Err(err).Str("profile", string(body)).Msg("profile can't be unmarshaled")
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s = sentry.StartSpan(ctx, "gcs.write")
	s.Description = "Write profile to GCS"
	_, err = storageutil.CompressedWrite(ctx, env.profilesBucket, profile.StoragePath(), body)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var p aggregate.Profile
	switch profile.Platform {
	case "cocoa":
		var cp aggregate.IosProfile
		s := sentry.StartSpan(ctx, "json.unmarshal")
		s.Description = "Unmarshal iOS profile"
		err := json.Unmarshal([]byte(profile.Profile), &cp)
		s.Finish()
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p = cp
	case "android":
		var ap android.AndroidProfile
		s := sentry.StartSpan(ctx, "json.unmarshal")
		s.Description = "Unmarshal Android profile"
		err := json.Unmarshal([]byte(profile.Profile), &ap)
		s.Finish()
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p = ap
	case "python", "rust", "node", "typescript":
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		hub.CaptureMessage("unknown platform")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s = sentry.StartSpan(ctx, "calltree")
	s.Description = "Generate call trees"
	callTrees := p.CallTrees()
	s.Finish()

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

func getRawProfile(ctx context.Context,
	organizationID uint64,
	projectID uint64,
	profileID string,
	profilesBucket *storage.BucketHandle,
	snuba snubautil.Client) (snubautil.Profile, error) {

	hub := sentry.GetHubFromContext(ctx)
	hub.Scope().SetTag("profile_id", profileID)

	var profile snubautil.Profile
	s := sentry.StartSpan(ctx, "gcs.read")
	s.Description = "Read profile from GCS"
	err := storageutil.UnmarshalCompressed(ctx, profilesBucket, snubautil.ProfileStoragePath(organizationID, projectID, profileID), &profile)
	s.Finish()
	if err != nil {
		if err != storage.ErrObjectNotExist {
			hub.CaptureException(err)
		}
		sqb, err := snuba.NewQuery(ctx, "profiles")
		if err != nil {
			hub.CaptureException(err)
			return snubautil.Profile{}, err
		}
		profile, err = snubautil.GetProfile(organizationID, projectID, profileID, sqb)
		if err != nil {
			if !errors.Is(err, snubautil.ErrProfileNotFound) {
				hub.CaptureException(err)
			}
			return snubautil.Profile{}, err
		}
	}
	return profile, nil
}
