package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"cloud.google.com/go/storage"
	"github.com/CAFxX/httpcompression"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/julienschmidt/httprouter"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"

	"github.com/getsentry/vroom/internal/httputil"
	"github.com/getsentry/vroom/internal/logutil"
	"github.com/getsentry/vroom/internal/snubautil"
)

type environment struct {
	Port           string `default:"8080"`
	ProfilesBucket string `envconfig:"SENTRY_PROFILES_BUCKET_NAME" required:"true"`
	SnubaHost      string `envconfig:"SENTRY_SNUBA_HOST" required:"true"`
	SnubaPort      string `envconfig:"SENTRY_SNUBA_PORT"`

	snuba snubautil.Client

	storage        *storage.Client
	profilesBucket *storage.BucketHandle
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
	e.storage, err = storage.NewClient(context.Background())
	if err != nil {
		return nil, err
	}
	e.profilesBucket = e.storage.Bucket(e.ProfilesBucket)
	return &e, nil
}

func (env *environment) shutdown() {
	err := env.storage.Close()
	if err != nil {
		sentry.CaptureException(err)
	}
	sentry.Flush(5 * time.Second)
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
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/functions", env.getFunctions},
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/profiles/:profile_id", env.getProfile},
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/raw_profiles/:profile_id", env.getRawProfile},
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/transactions/:transaction_id", env.getProfileIDByTransactionID},
		{http.MethodPost, "/profile", env.postProfile},
	}

	router := httprouter.New()

	for _, route := range routes {
		handlerFunc := httputil.AnonymizeTransactionName(route.handler)
		handlerFunc = httputil.DecompressPayload(handlerFunc)
		handler := compress(handlerFunc)

		router.Handler(route.method, route.path, handler)
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
	env.shutdown()
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
