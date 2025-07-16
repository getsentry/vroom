#!/bin/bash

checks-canary-canarychecks \
  --skip-check=${SKIP_CANARY_CHECKS} \
  --wait-minutes=5
