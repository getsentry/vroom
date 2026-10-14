#!/bin/bash

eval $(regions-project-env-vars --region="${SENTRY_REGION}")

# Run the distroless image variant in the regions below while rolling it out
# region by region. The "-distroless" tag is published by .github/workflows/image.yaml.
IMAGE_TAG="${GO_REVISION_VROOM_REPO}"
case "${SENTRY_REGION}" in
s4s2 | de | us)
	IMAGE_TAG="${GO_REVISION_VROOM_REPO}-distroless"
	;;
esac

/devinfra/scripts/get-cluster-credentials
k8s-deploy \
	--label-selector="${LABEL_SELECTOR}" \
	--image="us-central1-docker.pkg.dev/sentryio/vroom/vroom:${IMAGE_TAG}" \
	--container-name="vroom"
