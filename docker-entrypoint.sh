#!/bin/bash

set -euo pipefail

export PROFILES_DIR="${PROFILES_DIR:-/var/lib/sentry-profiles}"
export SENTRY_BUCKET_PROFILES="${SENTRY_BUCKET_PROFILES:-file://localhost/$PROFILES_DIR}"

su -
mkdir -p "$PROFILES_DIR"
chown -R vroom:vroom "$PROFILES_DIR"

su vroom
exec /bin/vroom "$@"
