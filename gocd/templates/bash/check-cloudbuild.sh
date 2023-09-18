#!/bin/bash

/devinfra/scripts/checks/googlecloud/checkcloudbuild.py \
	${GO_REVISION_VROOM_REPO} \
	"internal-sentry" \
	"us.gcr.io/internal-sentry/vroom"
