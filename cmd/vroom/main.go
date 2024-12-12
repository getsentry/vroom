package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CAFxX/httpcompression"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/julienschmidt/httprouter"
	"github.com/segmentio/kafka-go"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
	"gocloud.dev/gcerrors"

	"github.com/getsentry/vroom/internal/httputil"
	"github.com/getsentry/vroom/internal/logutil"
	"github.com/getsentry/vroom/internal/storageutil"
)

type environment struct {
	config ServiceConfig

	occurrencesWriter KafkaWriter
	profilingWriter   KafkaWriter

	storage *blob.Bucket
}

var (
	release  string
	readJobs chan storageutil.ReadJob
)

const (
	KiB int64 = 1024
	MiB       = 1024 * KiB
)

func newEnvironment() (*environment, error) {
	var e environment
	err := cleanenv.ReadEnv(&e.config)
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
		Transport:    createKafkaRoundTripper(e.config),
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
		Transport:    createKafkaRoundTripper(e.config),
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
			http.MethodPost,
			"/organizations/:organization_id/projects/:project_id/chunks",
			e.postProfileFromChunkIDs,
		},
		{
			http.MethodPost,
			"/organizations/:organization_id/flamegraph",
			e.postFlamegraph,
		},
		{
			http.MethodPost,
			"/organizations/:organization_id/metrics",
			e.postMetrics,
		},
		{http.MethodGet, "/health", e.getHealth},
		{http.MethodPost, "/chunk", e.postChunk},
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
		log.Fatal("error setting up environment", err)
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
		log.Fatal("can't initialize sentry", err)
	}

	router, err := env.newRouter()
	if err != nil {
		sentry.CaptureException(err)
		log.Fatal("error setting up the router", err)
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
			slog.Error("error shutting down server", "err", err)
		}

		close(waitForShutdown)
	}()

	slog.Info("vroom started")

	readJobs = make(chan storageutil.ReadJob, 10*env.config.WorkerPoolSize)
	for i := 0; i < env.config.WorkerPoolSize; i++ {
		go storageutil.ReadWorker(readJobs)
	}

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		sentry.CaptureException(err)
		slog.Error("server failed", "err", err)
	}

	<-waitForShutdown

	// Shutdown the rest of the environment after the HTTP connections are closed
	close(readJobs)
	env.shutdown()
	slog.Info("vroom graceful shutdown")
}

func (e *environment) getHealth(w http.ResponseWriter, _ *http.Request) {
	if _, err := os.Stat("/tmp/vroom.down"); err != nil {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadGateway)
	}
}
