#! /usr/bin/env bash

set -eou pipefail

export KAFKA_AUTO_CREATE_TOPICS_ENABLE="true"
export PORT="8085"
export SENTRY_DSN="https://91f2762536314cbd9cc4a163fe072682@o1.ingest.sentry.io/6424467"
export SENTRY_ENVIRONMENT="development"
export SENTRY_OCCURRENCES_ENABLED_ORGANIZATIONS="1:"
export SENTRY_PROFILES_BUCKET_NAME="sentry-profiles"
export SENTRY_SNUBA_HOST="http://localhost:1218"
export STORAGE_EMULATOR_HOST="http://0.0.0.0:8888/"

docker-compose -f ./deployments/docker-compose.yml up -d
./vroom
