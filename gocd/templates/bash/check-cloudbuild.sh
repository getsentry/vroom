#!/bin/bash

/devinfra/scripts/checks/googlecloud/check_cloudbuild.py \
	sentryio \
	vroom \
	build-vroom \
	${GO_REVISION_VROOM_REPO} \
	main
