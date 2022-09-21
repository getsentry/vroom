#! /usr/bin/env bash

set -euo pipefail

git_commit_id="$(git rev-parse HEAD)"

image="us-central1-docker.pkg.dev/specto-dev/vroom/vroom"

docker tag $image:latest $image:$git_commit_id
docker push $image:latest
docker push $image:$git_commit_id
