#! /usr/bin/env bash

set -eou pipefail

go build -o . -ldflags="-s -w" ./cmd/vroom
