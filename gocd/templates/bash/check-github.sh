#!/bin/bash

/devinfra/scripts/checks/githubactions/checkruns.py \
	getsentry/vroom \
	${GO_REVISION_VROOM_REPO} \
	test-vroom
