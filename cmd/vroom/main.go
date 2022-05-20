package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/CAFxX/httpcompression"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"

	"github.com/getsentry/vroom/internal/aggregate"
	"github.com/getsentry/vroom/internal/chrometrace"
	"github.com/getsentry/vroom/internal/httputil"
	"github.com/getsentry/vroom/internal/logutil"
	"github.com/getsentry/vroom/internal/snubautil"
)

type environment struct {
	Port      string `default:"8080"`
	SnubaHost string `envconfig:"SENTRY_PROFILING_SNUBA_HOST" required:"true"`
	SnubaPort int    `envconfig:"SENTRY_PROFILING_SNUBA_PORT" required:"false"`
}

func newEnvironment() (*environment, error) {
	var e environment
	err := envconfig.Process("", &e)
	if err != nil {
		log.Fatal().Err(err).Msg("organization: missing environment variables")
	}
	return &e, nil
}

func (env *environment) shutdown() error {
	return nil
}

func (env *environment) newRouter() (*httprouter.Router, error) {
	compress, err := httpcompression.DefaultAdapter()
	if err != nil {
		return nil, err
	}

	router := httprouter.New()

	router.Handler(http.MethodGet, "/organizations/:organization_id/filters", compress(http.HandlerFunc(env.getFilters)))
	router.Handler(http.MethodGet, "/organizations/:organization_id/profiles", compress(http.HandlerFunc(env.getProfiles)))
	router.Handler(http.MethodGet, "/organizations/:organization_id/transactions", compress(http.HandlerFunc(env.getTransactions)))
	router.Handler(http.MethodGet, "/organizations/:organization_id/projects/:project_id/functions_call_trees", compress(http.HandlerFunc(env.getFunctionsCallTrees)))
	router.Handler(http.MethodGet, "/organizations/:organization_id/projects/:project_id/functions_versions", compress(http.HandlerFunc(env.getFunctions)))
	router.Handler(http.MethodGet, "/organizations/:organization_id/projects/:project_id/profiles/:profile_id", compress(http.HandlerFunc(env.getProfile)))
	router.Handler(http.MethodGet, "/organizations/:organization_id/projects/:project_id/profiles/:profile_id/call_tree", compress(http.HandlerFunc(env.getProfileCallTree)))

	return router, nil
}

func main() {
	logutil.ConfigureLogger()

	env, err := newEnvironment()
	if err != nil {
		log.Fatal().Err(err).Msg("error setting up environment")
	}

	router, err := env.newRouter()
	if err != nil {
		log.Fatal().Err(err).Msg("error setting up the router")
	}

	server := http.Server{
		Addr:    ":" + env.Port,
		Handler: router,
	}

	waitForShutdown := make(chan os.Signal)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		cctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(cctx); err != nil {
			log.Err(err).Msg("error shutting down server")
		}

		close(waitForShutdown)
	}()

	log.Info().Str("port", env.Port).Msg("starting server")
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Err(err).Msg("server failed")
	}

	<-waitForShutdown

	// Shutdown the rest of the environment after the HTTP connections are closed
	if err := env.shutdown(); err != nil {
		log.Err(err).Msg("error tearing down environment")
	}
}

func (env *environment) getProfile(w http.ResponseWriter, r *http.Request) {
	ps := httprouter.ParamsFromContext(r.Context())
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		log.Err(err).Str("raw_organization_id", rawOrganizationID).Msg("invalid organization_id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rawProjectID := ps.ByName("project_id")
	projectID, err := strconv.ParseUint(rawProjectID, 10, 64)
	if err != nil {
		log.Err(err).Str("raw_project_id", rawProjectID).Msg("invalid project_id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	profileID := ps.ByName("profile_id")
	_, err = uuid.Parse(profileID)
	if err != nil {
		log.Err(err).Str("raw_profile_id", profileID).Msg("invalid profile_id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger := log.With().Uint64("organization_id", organizationID).Uint64("project_id", projectID).Str("profile_id", profileID).Logger()
	sqb := snubautil.SnubaQueryBuilder{
		Endpoint: env.SnubaHost,
		Port:     env.SnubaPort,
		Dataset:  "profiles",
		Entity:   "profiles",
	}
	profile, err := snubautil.GetProfile(organizationID, projectID, profileID, sqb)
	if err != nil {
		if errors.Is(err, snubautil.ErrProfileNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			logger.Err(err).Msg("cannot fetch profile data from snuba")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	logger = logger.With().Str("platform", profile.Platform).Logger()
	var b []byte
	switch profile.Platform {
	case "typescript", "javascript":
		b = []byte(profile.Profile)
	default:
		b, err = chrometrace.SpeedscopeFromSnuba(profile)
	}
	if err != nil {
		logger.Err(err).Msg("error creating chrome trace data")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

type GetOrganizationProfilesResponse struct {
	Profiles []ProfileResult `json:"profiles"`
}

type ProfileResult struct {
	AndroidAPILevel      uint32  `json:"android_api_level"`
	DeviceClassification string  `json:"device_classification"`
	DeviceLocale         string  `json:"device_locale"`
	DeviceManufacturer   string  `json:"device_manufacturer"`
	DeviceModel          string  `json:"device_model"`
	DeviceOsBuildNumber  string  `json:"device_os_build_number"`
	DeviceOsName         string  `json:"device_os_name"`
	DeviceOsVersion      string  `json:"device_os_version"`
	ID                   string  `json:"id"`
	ProjectID            string  `json:"project_id"`
	Timestamp            int64   `json:"timestamp"`
	TraceDurationMs      float64 `json:"trace_duration_ms"`
	TransactionID        string  `json:"transaction_id"`
	TransactionName      string  `json:"transaction_name"`
	VersionCode          string  `json:"version_code"`
	VersionName          string  `json:"version_name"`
}

func (env *environment) getProfiles(w http.ResponseWriter, r *http.Request) {
	ps := httprouter.ParamsFromContext(r.Context())
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		log.Err(err).
			Str("raw_organization_id", rawOrganizationID).
			Msg("organization_id path parameter is malformed and could not be parsed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, _, ok := httputil.GetRequiredQueryParameters(w, r, "project_id", "limit", "offset")
	if !ok {
		return
	}

	logger := log.With().Uint64("organization_id", organizationID).Logger()
	sqb, err := snubaQueryBuilderFromRequest(r.URL.Query())
	if err != nil {
		logger.Err(err).Msg("can't build snuba query from request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sqb.WhereConditions = append(sqb.WhereConditions, fmt.Sprintf("organization_id=%d", organizationID))
	sqb.Endpoint = env.SnubaHost
	sqb.Port = env.SnubaPort
	sqb.Dataset = "profiles"
	sqb.Entity = "profiles"

	profiles, err := snubautil.GetProfiles(sqb, false)
	if err != nil {
		logger.Err(err).Msg("error retrieving organization profiles")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := GetOrganizationProfilesResponse{
		Profiles: make([]ProfileResult, 0, len(profiles)),
	}

	for _, p := range profiles {
		resp.Profiles = append(resp.Profiles, snubaProfileToProfileResult(p))
	}

	b, err := json.Marshal(resp)
	if err != nil {
		logger.Err(err).Msg("error marshaling response to json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

type Filter struct {
	Key    string        `json:"key"`
	Name   string        `json:"name"`
	Values []interface{} `json:"values"`
}

func (env *environment) getFilters(w http.ResponseWriter, r *http.Request) {
	ps := httprouter.ParamsFromContext(r.Context())
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		log.Err(err).Str("raw_organization_id", rawOrganizationID).Msg("could not parse organization_id path parameter")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger := log.With().Uint64("organization_id", organizationID).Logger()
	sqb, err := snubaQueryBuilderFromRequest(r.URL.Query())
	if err != nil {
		logger.Err(err).Msg("can't build snuba query from request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sqb.WhereConditions = append(sqb.WhereConditions, fmt.Sprintf("organization_id = %d", organizationID))
	sqb.Endpoint = env.SnubaHost
	sqb.Port = env.SnubaPort
	sqb.Dataset = "profiles"
	sqb.Entity = "profiles"

	filters, err := snubautil.GetFilters(sqb)
	if err != nil {
		logger.Err(err).Msg("error retrieving organization profiles")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := make([]Filter, 0, len(filters))
	for k, v := range filters {
		response = append(response, Filter{Key: k, Name: k, Values: v})
	}

	b, err := json.Marshal(response)
	if err != nil {
		logger.Err(err).Msg("error marshaling response to json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

type GetFunctionsCallTreesResponse struct {
	FunctionCall aggregate.FunctionCall `json:"function_call"`
	CallTrees    []aggregate.CallTree   `json:"call_trees"`
}

func (env *environment) getFunctionsCallTrees(w http.ResponseWriter, r *http.Request) {
	ps := httprouter.ParamsFromContext(r.Context())
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		log.Err(err).
			Str("raw_organization_id", rawOrganizationID).
			Msg("organization_id path parameter is malformed and cannot be parsed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rawProjectID := ps.ByName("project_id")
	projectID, err := strconv.ParseUint(rawProjectID, 10, 64)
	if err != nil {
		log.Err(err).
			Str("raw_project_id", rawProjectID).
			Msg("project_id path parameter is malformed and cannot be parsed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	p, logger, ok := httputil.GetRequiredQueryParameters(w, r, "version", "transaction_name", "key")
	if !ok {
		return
	}
	logger.With().
		Uint64("organization_id", organizationID).
		Uint64("project_id", projectID).Logger()

	sqb, err := snubaQueryBuilderFromRequest(r.URL.Query())
	if err != nil {
		logger.Err(err).Msg("can't build snuba query from request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sqb.WhereConditions = append(sqb.WhereConditions,
		fmt.Sprintf("organization_id=%d", organizationID),
		fmt.Sprintf("project_id=%d", projectID),
	)
	sqb.Limit = 10

	sqb.Endpoint = env.SnubaHost
	sqb.Port = env.SnubaPort
	sqb.Dataset = "profiles"
	sqb.Entity = "profiles"

	profiles, err := snubautil.GetProfiles(sqb, true)
	if err != nil {
		logger.Err(err).Msg("error retrieving the profiles")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var topNFunctions int

	if topN, exists := r.URL.Query()["top_n_functions"]; exists && len(topN) == 1 {
		i, err := strconv.Atoi(topN[0])
		if err != nil {
			logger.Err(err).
				Str("top_n_functions", topN[0]).
				Msg("malformed query parameter cannot be parsed")
			w.WriteHeader(http.StatusBadRequest)
		} else {
			topNFunctions = i
		}
	}

	aggRes, err := aggregate.AggregateProfiles(profiles, topNFunctions)
	if err != nil {
		logger.Err(err).Msg("aggregation: error while trying to compute the aggregation")
	}

	var response GetFunctionsCallTreesResponse
	// Linear search the list of functions for the one matching the key
	// because N is small (less than 100 elements).
	for _, f := range aggRes.Aggregation.FunctionCalls {
		if f.Key == p["key"] {
			response.FunctionCall = f
			break
		}
	}
	if trees, ok := aggRes.Aggregation.FunctionToCallTrees[p["key"]]; ok {
		aggregate.RemoveDurationValuesFromCallTreesP(trees)
		response.CallTrees = trees
	}
	if len(response.CallTrees) == 0 {
		logger.Error().Msg("no call trees")
	}
	b, err := json.Marshal(response)
	if err != nil {
		logger.Err(err).Msg("error marshaling response to json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

type FunctionCallsData struct {
	FunctionCalls []aggregate.FunctionCall
}

type VersionSeriesData struct {
	Versions map[string]FunctionCallsData
}

func (env *environment) getFunctions(w http.ResponseWriter, r *http.Request) {
	ps := httprouter.ParamsFromContext(r.Context())
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		log.Err(err).
			Str("raw_organization_id", rawOrganizationID).
			Msg("organization_id path parameter is malformed and cannot be parsed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rawProjectID := ps.ByName("project_id")
	projectID, err := strconv.ParseUint(rawProjectID, 10, 64)
	if err != nil {
		log.Err(err).
			Str("raw_project_id", rawProjectID).
			Msg("project_id path parameter is malformed and cannot be parsed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, logger, ok := httputil.GetRequiredQueryParameters(w, r, "version", "transaction_name")
	if !ok {
		return
	}
	logger.With().
		Uint64("organization_id", organizationID).
		Uint64("project_id", projectID).Logger()

	var topNFunctions int

	if topN, exists := r.URL.Query()["top_n_functions"]; exists && len(topN) == 1 {
		i, err := strconv.Atoi(topN[0])
		if err != nil {
			logger.Err(err).
				Str("top_n_functions", topN[0]).
				Msg("malformed query parameter cannot be parsed")
			w.WriteHeader(http.StatusBadRequest)
		} else {
			topNFunctions = i
		}
	}

	sqb, err := snubaQueryBuilderFromRequest(r.URL.Query())
	if err != nil {
		logger.Err(err).Msg("can't build snuba query from request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// placeholder is a condition we'll replace each time with the specific app_version
	// for which we want to fetch profiles
	sqb.WhereConditions = append(sqb.WhereConditions,
		fmt.Sprintf("organization_id=%d", organizationID),
		fmt.Sprintf("project_id=%d", projectID),
		"placeholder",
	)
	sqb.Limit = 10

	sqb.Endpoint = env.SnubaHost
	sqb.Port = env.SnubaPort
	sqb.Dataset = "profiles"
	sqb.Entity = "profiles"

	// here it is safe to ignore the errors because if there were any, they'd be
	// already triggered when calling snubaQueryBuilderFromRequest
	versionBuilds, _ := GetVersionBuildFromAppVersions(r.URL.Query()["version"])
	versionToProfiles := make(map[string][]snubautil.Profile)
	for _, versionBuild := range versionBuilds {
		// replace the placeholder condition
		sqb.WhereConditions[len(sqb.WhereConditions)-1] = fmt.Sprintf("(version_name = '%s' AND version_code = '%s')", snubautil.Escape(versionBuild.Name), snubautil.Escape(versionBuild.Code))
		profiles, err := snubautil.GetProfiles(sqb, true)
		if err != nil {
			logger.Err(err).Msg("error retrieving the profiles")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		versionToProfiles[fmt.Sprintf("%s (build %s)", versionBuild.Name, versionBuild.Code)] = profiles
	}

	versionMap := make(map[string]FunctionCallsData)
	for version, profiles := range versionToProfiles {

		aggResult, err := aggregate.AggregateProfiles(profiles, topNFunctions)
		if err != nil {
			logger.Err(err).
				Str("version", version).
				Msg("error while running the aggregation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		versionMap[version] = FunctionCallsData{
			FunctionCalls: aggResult.Aggregation.FunctionCalls,
		}
	}

	versionData := VersionSeriesData{
		Versions: versionMap,
	}

	b, err := json.Marshal(versionData)
	if err != nil {
		logger.Err(err).Msg("error marshaling response to json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
