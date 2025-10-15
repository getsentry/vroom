#!/bin/bash

checks-githubactions-checkruns \
	getsentry/vroom \
	${GO_REVISION_VROOM_REPO} \
	test-vroom \
	'Build and push production images'
