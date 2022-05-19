#! /usr/bin/env bash

set -euo pipefail

image="us-central1-docker.pkg.dev/specto-dev/vroom/vroom"

docker push $image
