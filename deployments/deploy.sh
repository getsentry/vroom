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
  --set-env-vars=SENTRY_DSN=https://91f2762536314cbd9cc4a163fe072682@o1.ingest.sentry.io/6424467 \
  --set-env-vars=SENTRY_ENVIRONMENT=production \
  --set-env-vars=SENTRY_PROFILES_BUCKET_NAME=sentry-profiles \
  --set-env-vars=SENTRY_RELEASE="$git_commit_id" \
  --set-env-vars=SENTRY_SNUBA_HOST=http://snuba-api.profiling \
  --set-env-vars=SENTRY_OCCURRENCES_ENABLED_ORGANIZATIONS="1:,447951:" \
  --set-env-vars=SENTRY_OCCURRENCES_KAFKA_BROKERS="specto-dev-kafka.service.us-central1.consul:9092" \
  --set-env-vars=SENTRY_OCCURRENCES_KAFKA_TOPIC=ingest-occurrences \
  --timeout 30s \
  --vpc-connector sentry-ingest
