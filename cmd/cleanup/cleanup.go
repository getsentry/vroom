package main

import (
	"errors"
	"os"
	"os/signal"
	"path"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/logutil"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

func cleanup(profilesPath string, timeLimit time.Time) error {
	dirEntries, err := os.ReadDir(profilesPath)
	if err != nil {
		return err
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			return cleanup(path.Join(profilesPath, entry.Name()), timeLimit)
		}

		fileInfo, err := entry.Info()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}

		if timeLimit.After(fileInfo.ModTime()) {
			err = os.Remove(path.Join(profilesPath, entry.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	sentryBucketProfiles, ok := os.LookupEnv("SENTRY_BUCKET_PROFILES")
	if !ok {
		sentryBucketProfiles = "/var/lib/sentry-profiles"
	}

	sentryEventRetentionDays, ok := os.LookupEnv("SENTRY_EVENT_RETENTION_DAYS")
	if !ok {
		sentryEventRetentionDays = "90"
	}

	logutil.ConfigureLogger()

	err := sentry.Init(sentry.ClientOptions{})
	if err != nil {
		log.Fatal().Err(err).Msg("can't initialize sentry")
	}

	retentionDays, err := strconv.ParseInt(sentryEventRetentionDays, 10, 64)
	if err != nil {
		log.Fatal().Err(err).Msg("can't parse retention days")
	}

	timeLimit := time.Now().Add(time.Hour * 24 * -1 * time.Duration(retentionDays))

	c := cron.New()
	_, err = c.AddFunc("@daily", func() {
		err := cleanup(sentryBucketProfiles, timeLimit)
		if err != nil {
			sentry.CaptureException(err)
			log.Error().Err(err).Msg("error cleaning up directories")
		}
	})
	if err != nil {
		log.Fatal().Err(err).Msg("can't set up cron function")
	}

	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, os.Interrupt)

	go func() {
		<-exitSignal

		c.Stop()
	}()

	c.Run()
}
