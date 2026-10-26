#! /usr/bin/env bash

set -eou pipefail

export PORT="8085"
export STORAGE_EMULATOR_HOST="http://0.0.0.0:8888/"

./vroom
