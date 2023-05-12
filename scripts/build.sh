#! /usr/bin/env bash

# EXec
set -eou pipefail

go build -o . -ldflags="-s -w" ./cmd/vroom
