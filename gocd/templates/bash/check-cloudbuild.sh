#!/bin/bash

/devinfra/scripts/checks/googlecloud/checkcloudbuild.py \
	${GO_REVISION_VROOM_REPO} \
	${GCP_PROJECT} \
	"us.gcr.io/${GCP_PROJECT}/vroom"
