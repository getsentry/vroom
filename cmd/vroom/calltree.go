package main

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"github.com/getsentry/vroom/internal/aggregate"
	"github.com/getsentry/vroom/internal/snubautil"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/log"
)

type GetProfileCallTreeResponse struct {
	CallTrees []aggregate.CallTree `json:"call_trees"`
}

func (env *environment) getProfileCallTree(w http.ResponseWriter, r *http.Request) {
	ps := httprouter.ParamsFromContext(r.Context())
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		log.Err(err).
			Str("raw_organization_id", rawOrganizationID).
			Msg("aggregate: organization_id path parameter is malformed and cannot be parsed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rawProjectID := ps.ByName("project_id")
	projectID, err := strconv.ParseUint(rawProjectID, 10, 64)
	if err != nil {
		log.Err(err).
			Str("raw_project_id", rawProjectID).
			Msg("aggregate: project_id path parameter is malformed and cannot be parsed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	profileID := ps.ByName("profile_id")
	logger := log.With().Uint64("organization_id", organizationID).Uint64("project_id", projectID).Str("profile_id", profileID).Logger()
	sqb := snubautil.SnubaQueryBuilder{
		Endpoint: env.SnubaHost,
		Port:     env.SnubaPort,
		Dataset:  "profiles",
		Entity:   "profiles",
	}

	profiles, err := snubautil.GetProfile(organizationID, projectID, profileID, sqb)
	if err != nil {
		logger.Err(err).Msg("aggregate: error retrieving the profiles")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	aggRes, err := aggregate.AggregateProfiles([]snubautil.Profile{profiles}, math.MaxInt)
	if err != nil {
		logger.Err(err).Msg("aggregation: error while trying to compute the aggregation")
	}

	merged, err := aggregate.MergeAllCallTreesInBacktrace(&aggRes.Aggregation)
	if err != nil {
		log.Error().Err(err).Msg("aggregate: error merging single trace aggregation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(GetProfileCallTreeResponse{
		CallTrees: merged,
	})
	if err != nil {
		logger.Err(err).Msg("aggregate: error marshaling response to json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
