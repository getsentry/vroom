#! /usr/bin/env bash

set -euo pipefail

git_commit_id="$(git rev-parse HEAD)"
image="us-central1-docker.pkg.dev/specto-dev/vroom/vroom:$git_commit_id"

gcloud beta run deploy vroom \
  --concurrency 10 \
  --cpu 1 \
  --execution-environment gen2 \
  --image $image \
  --memory 1Gi \
  --allow-unauthenticated \
  --ingress internal-and-cloud-load-balancing \
  --vpc-egress all-traffic \
  --region us-central1 \
  --service-account service-vroom@specto-dev.iam.gserviceaccount.com \
  --set-env-vars=SENTRY_ENVIRONMENT=production \
  --timeout 30s \
  --vpc-connector sentry-ingest
