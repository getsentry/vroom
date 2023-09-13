#!/bin/bash

/devinfra/scripts/canary/canarychecks.py \
  --skip-check=${SKIP_CANARY_CHECKS} \
  --wait-minutes=5
