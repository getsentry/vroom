#! /usr/bin/env bash

set -eou pipefail

export SENTRY_PROFILING_SNUBA_HOST="http://localhost:1218"
export PORT="8085"

./vroom
