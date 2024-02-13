package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/CAFxX/httpcompression"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
	"gocloud.dev/gcerrors"

	"github.com/getsentry/vroom/internal/httputil"
	"github.com/getsentry/vroom/internal/logutil"
	"github.com/getsentry/vroom/internal/snubautil"
)

type environment struct {
	config  ServiceConfig
	options atomic.Value
	snuba   snubautil.Client

	occurrencesWriter *kafka.Writer
	profilingWriter   *kafka.Writer

	storage *blob.Bucket
}

var release string

const (
	KiB                int64 = 1024
	MiB                      = 1024 * KiB
	OptionsPollingSecs       = 20
)

func newEnvironment() (*environment, error) {
	var e environment
	err := cleanenv.ReadEnv(&e.config)
	if err != nil {
		return nil, err
	}

	e.snuba, err = snubautil.NewClient(e.config.SnubaHost, "profiles")
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	e.storage, err = blob.OpenBucket(ctx, e.config.BucketURL)
	if err != nil {
		return nil, err
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
		BatchBytes:   20 * MiB,
		BatchSize:    10,
		Compression:  kafka.Lz4,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
	options, err := getSentryOptions(e.config.SentryHost)
	// if getSentryOptions succeed we store the options, otherwise
	// we just ignore it to avoid failing the whole vroom bootstrap.
	if err == nil {
		e.options.Store(options)
	}
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
		{
			http.MethodGet,
			"/organizations/:organization_id/projects/:project_id/functions",
			e.getFunctions,
		},
		{
			http.MethodGet,
			"/organizations/:organization_id/projects/:project_id/profiles/:profile_id",
			e.getProfile,
		},
		{
			http.MethodGet,
			"/organizations/:organization_id/projects/:project_id/raw_profiles/:profile_id",
			e.getRawProfile,
		},
		{
			http.MethodGet,
			"/organizations/:organization_id/projects/:project_id/transactions/:transaction_id",
			e.getProfileIDByTransactionID,
		},
		{
			http.MethodGet,
			"/organizations/:organization_id/projects/:project_id/flamegraph",
			e.getFlamegraph,
		},
		{
			http.MethodPost,
			"/organizations/:organization_id/projects/:project_id/flamegraph",
			e.postFlamegraphFromProfileIDs,
		},
		{http.MethodGet, "/health", e.getHealth},
		{http.MethodPost, "/profile", e.postProfile},
		{http.MethodPost, "/regressed", e.postRegressed},
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
		Dsn:                   env.config.SentryDSN,
		EnableTracing:         true,
		TracesSampleRate:      1.0,
		Environment:           env.config.Environment,
		Release:               release,
		BeforeSendTransaction: httputil.SetHTTPStatusCodeTag,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			code := gcerrors.Code(hint.OriginalException)
			switch code {
			// Ignore unknown or network errors as gcerrors returns a specific GCS error
			// in case we have generic network errors, even if it didn't come from the gocloud
			// library and we can't check for the specific gocloud error type as it's in
			// an internal package.
			case gcerrors.Canceled, gcerrors.DeadlineExceeded, gcerrors.Unknown, gcerrors.OK:
			default:
				event.Fingerprint = []string{"{{ default }}", code.String()}
			}
			return event
		},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("can't initialize sentry")
	}

	router, err := env.newRouter()
	if err != nil {
		sentry.CaptureException(err)
		log.Fatal().Err(err).Msg("error setting up the router")
	}

	server := http.Server{
		Addr:              fmt.Sprintf(":%d", env.config.Port),
		ReadHeaderTimeout: time.Second,
		Handler:           sentryhttp.New(sentryhttp.Options{}).Handle(router),
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

	ticker := time.NewTicker(OptionsPollingSecs * time.Second)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-waitForShutdown:
				return
			case <-ticker.C:
				opt, err := getSentryOptions(env.config.SentryHost)
				if err != nil {
					sentry.CaptureException(err)
				}
				env.options.Store(opt)
			}
		}
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

func (e *environment) getHealth(w http.ResponseWriter, _ *http.Request) {
	if _, err := os.Stat("/tmp/vroom.down"); err != nil {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadGateway)
	}
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

	sqb, err := e.profilesQueryBuilderFromRequest(ctx, r.URL.Query(), organizationID)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sqb.WhereConditions = append(
		sqb.WhereConditions,
		fmt.Sprintf("organization_id = %d", organizationID),
	)

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
