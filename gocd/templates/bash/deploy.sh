#!/bin/bash

eval $(regions-project-env-vars --region="${SENTRY_REGION}")

/devinfra/scripts/k8s/k8stunnel
k8s-deploy \
	--label-selector="${LABEL_SELECTOR}" \
	--image="us-central1-docker.pkg.dev/sentryio/vroom/vroom:${GO_REVISION_VROOM_REPO}" \
	--container-name="vroom"
