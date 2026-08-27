#!/bin/bash

checks-githubactions-checkruns2 \
	getsentry/vroom \
	${GO_REVISION_VROOM_REPO} \
	test-vroom \
	'Build and push production images'
