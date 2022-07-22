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
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
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
	SnubaHost string `envconfig:"SENTRY_SNUBA_HOST" required:"true"`
	SnubaPort string `envconfig:"SENTRY_SNUBA_PORT"`

	snuba snubautil.Client
}

func newEnvironment() (*environment, error) {
	var e environment
	err := envconfig.Process("", &e)
	if err != nil {
		return nil, err
	}
	e.snuba, err = snubautil.NewClient(e.SnubaHost, e.SnubaPort, "profiles", sentry.CurrentHub())
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (env *environment) shutdown() error {
	sentry.Flush(5 * time.Second)
	return nil
}

func (env *environment) newRouter() (*httprouter.Router, error) {
	compress, err := httpcompression.DefaultAdapter()
	if err != nil {
		return nil, err
	}

	routes := []struct {
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{http.MethodGet, "/organizations/:organization_id/filters", env.getFilters},
		{http.MethodGet, "/organizations/:organization_id/profiles", env.getProfiles},
		{http.MethodGet, "/organizations/:organization_id/transactions", env.getTransactions},
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/functions_call_trees", env.getFunctionsCallTrees},
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/functions_versions", env.getFunctions},
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/profiles/:profile_id", env.getProfile},
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/profiles/:profile_id/call_tree", env.getProfileCallTree},
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/transactions/:transaction_id", env.getProfileIDByTransactionID},
		{http.MethodPost, "/call_tree", env.postCallTree},
	}

	router := httprouter.New()

	for _, route := range routes {
		router.Handler(route.method, route.path, compress(httputil.AnonymizeTransactionName(http.HandlerFunc(route.handler))))
	}

	return router, nil
}

func main() {
	logutil.ConfigureLogger()

	err := sentry.Init(sentry.ClientOptions{
		TracesSampleRate: 1.0,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("can't initialize sentry")
	}

	env, err := newEnvironment()
	if err != nil {
		sentry.CaptureException(err)
		log.Fatal().Err(err).Msg("error setting up environment")
	}

	router, err := env.newRouter()
	if err != nil {
		sentry.CaptureException(err)
		log.Fatal().Err(err).Msg("error setting up the router")
	}

	server := http.Server{
		Addr:    ":" + env.Port,
		Handler: sentryhttp.New(sentryhttp.Options{}).Handle(router),
	}

	waitForShutdown := make(chan os.Signal)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		cctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(cctx); err != nil {
			sentry.CaptureException(err)
			log.Err(err).Msg("error shutting down server")
		}

		close(waitForShutdown)
	}()

	log.Info().Str("port", env.Port).Msg("starting server")
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		sentry.CaptureException(err)
		log.Err(err).Msg("server failed")
	}

	<-waitForShutdown

	// Shutdown the rest of the environment after the HTTP connections are closed
	if err := env.shutdown(); err != nil {
		sentry.CaptureException(err)
		log.Err(err).Msg("error tearing down environment")
	}
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

	hub.Scope().SetTag("platform", profile.Platform)

	s := sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()

	var b []byte
	switch profile.Platform {
	case "typescript", "javascript":
		b = []byte(profile.Profile)
	default:
		b, err = chrometrace.SpeedscopeFromSnuba(profile)
	}
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
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)
	_, ok := httputil.GetRequiredQueryParameters(w, r, "project_id", "limit", "offset")
	if !ok {
		return
	}

	ps := httprouter.ParamsFromContext(ctx)
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("organization_id", rawOrganizationID)

	sqb, err := env.profilesQueryBuilderFromRequest(ctx, r.URL.Query())
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sqb.WhereConditions = append(sqb.WhereConditions, fmt.Sprintf("organization_id=%d", organizationID))

	profiles, err := snubautil.GetProfiles(sqb, false)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s := sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()

	resp := GetOrganizationProfilesResponse{
		Profiles: make([]ProfileResult, 0, len(profiles)),
	}

	for _, p := range profiles {
		resp.Profiles = append(resp.Profiles, snubaProfileToProfileResult(p))
	}

	b, err := json.Marshal(resp)
	if err != nil {
		hub.CaptureException(err)
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

	sqb, err := env.profilesQueryBuilderFromRequest(ctx, r.URL.Query())
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sqb.WhereConditions = append(sqb.WhereConditions, fmt.Sprintf("organization_id = %d", organizationID))

	filters, err := snubautil.GetFilters(sqb)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s := sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()

	response := make([]Filter, 0, len(filters))
	for k, v := range filters {
		response = append(response, Filter{Key: k, Name: k, Values: v})
	}

	b, err := json.Marshal(response)
	if err != nil {
		hub.CaptureException(err)
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
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)
	p, ok := httputil.GetRequiredQueryParameters(w, r, "version", "transaction_name", "key")
	if !ok {
		return
	}

	hub.Scope().SetTags(p)

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

	hub.Scope().SetTag("project_id", rawProjectID)

	sqb, err := env.profilesQueryBuilderFromRequest(ctx, r.URL.Query())
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sqb.Limit = 10
	sqb.WhereConditions = append(sqb.WhereConditions,
		fmt.Sprintf("organization_id=%d", organizationID),
		fmt.Sprintf("project_id=%d", projectID),
	)

	profiles, err := snubautil.GetProfiles(sqb, true)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var topNFunctions int
	if rawTopNFunctions := r.URL.Query().Get("top_n_functions"); rawTopNFunctions != "" {
		i, err := strconv.Atoi(rawTopNFunctions)
		if err != nil {
			sentry.CaptureException(err)
			w.WriteHeader(http.StatusBadRequest)
		} else {
			topNFunctions = i
		}
	}

	s := sentry.StartSpan(ctx, "aggregation")
	aggRes, err := aggregate.AggregateProfiles(profiles, topNFunctions)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()

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
		response.CallTrees = trees
	}

	if len(response.CallTrees) == 0 {
		hub.CaptureMessage("no call tree")
	}

	b, err := json.Marshal(response)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

type (
	GetFunctionsResponse struct {
		Functions []aggregate.FunctionCall `json:"functions"`
	}
)

func (env *environment) getFunctions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)
	p, ok := httputil.GetRequiredQueryParameters(w, r, "transaction_name")
	if !ok {
		return
	}

	hub.Scope().SetTags(p)

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

	hub.Scope().SetTag("project_id", rawProjectID)

	var topNFunctions int
	if rawTopNFunctions := r.URL.Query().Get("top_n_functions"); rawTopNFunctions != "" {
		i, err := strconv.Atoi(rawTopNFunctions)
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusBadRequest)
		} else {
			topNFunctions = i
		}
	}

	sqb, err := env.profilesQueryBuilderFromRequest(ctx, r.URL.Query())
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sqb.WhereConditions = append(sqb.WhereConditions,
		fmt.Sprintf("organization_id=%d", organizationID),
		fmt.Sprintf("project_id=%d", projectID),
	)
	sqb.Limit = 10

	profiles, err := snubautil.GetProfiles(sqb, true)
	if err != nil {
		hub.CaptureException(err)
		return
	}

	s := sentry.StartSpan(ctx, "aggregation")
	aggResult, err := aggregate.AggregateProfiles(profiles, topNFunctions)
	s.Finish()
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s = sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()

	b, err := json.Marshal(GetFunctionsResponse{
		Functions: aggResult.Aggregation.FunctionCalls,
	})
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
