package main

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/aggregate"
	"github.com/getsentry/vroom/internal/snubautil"
	"github.com/julienschmidt/httprouter"
)

type GetProfileCallTreeResponse struct {
	CallTrees []aggregate.CallTree `json:"call_trees"`
}

func (env *environment) getProfileCallTree(w http.ResponseWriter, r *http.Request) {
	hub := sentry.GetHubFromContext(r.Context())
	ps := httprouter.ParamsFromContext(r.Context())
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		hub.Scope().SetContext("raw_organization_id", rawOrganizationID)
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("organization_id", rawOrganizationID)

	rawProjectID := ps.ByName("project_id")
	projectID, err := strconv.ParseUint(rawProjectID, 10, 64)
	if err != nil {
		hub.Scope().SetContext("raw_project_id", rawProjectID)
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	profileID := ps.ByName("profile_id")

	hub.Scope().SetTags(map[string]string{
		"project_id": rawProjectID,
		"profile_id": profileID,
	})

	sqb := snubautil.SnubaQueryBuilder{
		Endpoint: env.SnubaHost,
		Port:     env.SnubaPort,
		Dataset:  "profiles",
		Entity:   "profiles",
	}
	profiles, err := snubautil.GetProfile(organizationID, projectID, profileID, sqb)
	if err != nil {
		if errors.Is(err, snubautil.ErrProfileNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			q, _ := sqb.Query()
			hub.Scope().SetContext("query", q)
			hub.CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	aggRes, err := aggregate.AggregateProfiles([]snubautil.Profile{profiles}, math.MaxInt)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	merged, err := aggregate.MergeAllCallTreesInBacktrace(&aggRes.Aggregation)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

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
