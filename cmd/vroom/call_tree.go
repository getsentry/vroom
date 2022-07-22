package main

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/aggregate"
	"github.com/getsentry/vroom/internal/android"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/snubautil"
	"github.com/julienschmidt/httprouter"
)

type GetProfileCallTreeResponse struct {
	CallTrees []aggregate.CallTree `json:"call_trees"`
}

func (env *environment) getProfileCallTree(w http.ResponseWriter, r *http.Request) {
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
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	profileID := ps.ByName("profile_id")

	hub.Scope().SetTags(map[string]string{
		"project_id": rawProjectID,
		"profile_id": profileID,
	})

	sqb, err := env.snuba.NewQuery(ctx, "profiles")
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	profile, err := snubautil.GetProfile(organizationID, projectID, profileID, sqb)
	if err != nil {
		if errors.Is(err, snubautil.ErrProfileNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	s := sentry.StartSpan(ctx, "aggregation")
	s.Description = "Aggregate profiles"
	aggRes, err := aggregate.AggregateProfiles([]snubautil.Profile{profile}, math.MaxInt)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s = sentry.StartSpan(ctx, "merge")
	s.Description = "Merge all call trees in one"
	merged, err := aggregate.MergeAllCallTreesInBacktrace(&aggRes.Aggregation)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()

	b, err := json.Marshal(GetProfileCallTreeResponse{
		CallTrees: merged,
	})
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

type PostCallTreeResponse struct {
	CallTrees map[uint64][]*nodetree.Node `json:"call_trees"`
}

func (env *environment) postCallTree(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)

	s := sentry.StartSpan(ctx, "json.unmarshal")
	s.Description = "Unmarshal Snuba profile"
	var profile snubautil.Profile
	err := json.NewDecoder(r.Body).Decode(&profile)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
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

	b, err := json.Marshal(PostCallTreeResponse{
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
