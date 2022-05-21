#! /usr/bin/env bash

set -eou pipefail

export SENTRY_PROFILING_SNUBA_HOST="http://localhost:1218"
export SENTRY_ENVIRONMENT="development"
export SENTRY_DSN="https://91f2762536314cbd9cc4a163fe072682@o1.ingest.sentry.io/6424467"
export PORT="8085"

./vroom
