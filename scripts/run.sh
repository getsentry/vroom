#! /usr/bin/env bash

set -eou pipefail

export PORT="8085"
export SENTRY_DSN="https://91f2762536314cbd9cc4a163fe072682@o1.ingest.sentry.io/6424467"
export SENTRY_ENVIRONMENT="development"
export SENTRY_PROFILES_BUCKET_NAME="sentry-profiles"
export SENTRY_SNUBA_HOST="http://localhost:1218"

./vroom
