#!/bin/bash

/devinfra/scripts/checks/googlecloud/checkcloudbuild.py \
	${GO_REVISION_VROOM_REPO} \
	"sentryio" \
	"us-central1-docker.pkg.dev/sentryio/vroom"
