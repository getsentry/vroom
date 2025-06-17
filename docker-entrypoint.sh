#!/bin/bash

set -euo pipefail

export PROFILES_DIR="${PROFILES_DIR:-/var/lib/sentry-profiles}"
export SENTRY_BUCKET_PROFILES="${SENTRY_BUCKET_PROFILES:-file://localhost/$PROFILES_DIR}"

mkdir -p "$PROFILES_DIR"

if [ $(id -u) -eq 0 ]; then
  echo "Running as root, trying to get ownership of $PROFILES_DIR"
  chown -R vroom:vroom "$PROFILES_DIR"
  echo "Switching to user vroom"
  su vroom
fi

exec /bin/vroom "$@"
