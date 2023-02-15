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

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/CAFxX/httpcompression"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"

	"github.com/getsentry/vroom/internal/httputil"
	"github.com/getsentry/vroom/internal/logutil"
	"github.com/getsentry/vroom/internal/snubautil"
)

type environment struct {
	config ServiceConfig

	snuba snubautil.Client

	occurrencesWriter   *kafka.Writer
	profilingWriter     *kafka.Writer
	occurrencesInserter *bigquery.Inserter

	storage        *storage.Client
	profilesBucket *storage.BucketHandle
}

var release string

func newEnvironment() (*environment, error) {
	envName := os.Getenv("SENTRY_ENVIRONMENT")
	if envName == "" {
		envName = "development"
	}
	var e environment
	var exists bool
	e.config, exists = serviceConfigs[envName]
	if !exists {
		return nil, fmt.Errorf("service config for environment %v does not exist", envName)
	}
	e.config.Environment = envName

	var err error
	e.snuba, err = snubautil.NewClient(e.config.SnubaHost, "profiles", sentry.CurrentHub())
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	e.storage, err = storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	if envName == "production" {
		bqClient, err := bigquery.NewClient(ctx, "specto-dev")
		if err != nil {
			return nil, err
		}
		e.occurrencesInserter = bqClient.Dataset("issues").Table("occurrences").Inserter()
	}
	e.occurrencesWriter = &kafka.Writer{
		Addr:         kafka.TCP(e.config.OccurrencesKafkaBrokers...),
		Async:        true,
		Balancer:     kafka.CRC32Balancer{},
		BatchSize:    100,
		ReadTimeout:  3 * time.Second,
		Topic:        e.config.OccurrencesKafkaTopic,
		WriteTimeout: 3 * time.Second,
	}
	e.profilingWriter = &kafka.Writer{
		Addr:         kafka.TCP(e.config.ProfilingKafkaBrokers...),
		Async:        true,
		Balancer:     kafka.CRC32Balancer{},
		BatchSize:    10,
		Compression:  kafka.Lz4,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
	e.profilesBucket = e.storage.Bucket(e.config.ProfilesBucket)
	return &e, nil
}

func (e *environment) shutdown() {
	err := e.storage.Close()
	if err != nil {
		sentry.CaptureException(err)
	}
	err = e.occurrencesWriter.Close()
	if err != nil {
		sentry.CaptureException(err)
	}
	err = e.profilingWriter.Close()
	if err != nil {
		sentry.CaptureException(err)
	}
	sentry.Flush(5 * time.Second)
}

func (e *environment) newRouter() (*httprouter.Router, error) {
	compress, err := httpcompression.DefaultAdapter()
	if err != nil {
		return nil, err
	}

	routes := []struct {
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{http.MethodGet, "/organizations/:organization_id/filters", e.getFilters},
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/functions", e.getFunctions},
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/profiles/:profile_id", e.getProfile},
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/raw_profiles/:profile_id", e.getRawProfile},
		{http.MethodGet, "/organizations/:organization_id/projects/:project_id/transactions/:transaction_id", e.getProfileIDByTransactionID},
		{http.MethodGet, "/health", e.getHealth},
		{http.MethodPost, "/profile", e.postProfile},
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

	env, err := newEnvironment()
	if err != nil {
		log.Fatal().Err(err).Msg("error setting up environment")
	}

	err = sentry.Init(sentry.ClientOptions{
		Dsn:              env.config.SentryDSN,
		EnableTracing:    true,
		Environment:      env.config.Environment,
		Release:          release,
		TracesSampleRate: 1.0,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("can't initialize sentry")
	}

	router, err := env.newRouter()
	if err != nil {
		sentry.CaptureException(err)
		log.Fatal().Err(err).Msg("error setting up the router")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := http.Server{
		Addr:    ":" + port,
		Handler: sentryhttp.New(sentryhttp.Options{}).Handle(router),
	}

	waitForShutdown := make(chan os.Signal)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		cctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(cctx); err != nil {
			sentry.CaptureException(err)
			log.Err(err).Msg("error shutting down server")
		}

		close(waitForShutdown)
	}()

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		sentry.CaptureException(err)
		log.Err(err).Msg("server failed")
	}

	<-waitForShutdown

	// Shutdown the rest of the environment after the HTTP connections are closed
	env.shutdown()
}

func (e *environment) getHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

type Filter struct {
	Key    string        `json:"key"`
	Name   string        `json:"name"`
	Values []interface{} `json:"values"`
}

func (e *environment) getFilters(w http.ResponseWriter, r *http.Request) {
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

	sqb, err := e.profilesQueryBuilderFromRequest(ctx, r.URL.Query())
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
