#! /usr/bin/env bash

set -euo pipefail

image="us-central1-docker.pkg.dev/specto-dev/vroom/vroom:latest"

gcloud beta run deploy vroom \
  --concurrency 10 \
  --cpu 1 \
  --execution-environment gen2 \
  --image $image \
  --memory 1Gi \
  --no-allow-unauthenticated \
  --region us-central1 \
  --service-account service-vroom@specto-dev.iam.gserviceaccount.com \
  --set-env-vars=SENTRY_PROFILING_SNUBA_HOST=http://snuba-api.profiling \
  --timeout 30s \
  --vpc-connector sentry-ingest
