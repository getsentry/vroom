#! /usr/bin/env bash

set -eou pipefail

export ENVIRONMENT="development"
export KAFKA_AUTO_CREATE_TOPICS_ENABLE="true"
export PORT="8085"
export STORAGE_EMULATOR_HOST="http://0.0.0.0:8888/"

docker-compose -f ./deployments/docker-compose.yml up -d
./vroom
