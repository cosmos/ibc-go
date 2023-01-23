#!/bin/bash

set -euo pipefail

ENTRY_POINT="${1}"
TEST="${2}"

export CHAIN_A_TAG="${CHAIN_A_TAG:-latest}"
export CHAIN_IMAGE="${CHAIN_IMAGE:-ibc-go-simd}"
export CHAIN_BINARY="${CHAIN_BINARY:-simd}"

# In CI, the docker images will be built separately.
# context for building the image is one directory up.
if [ "${CI:-}" != "true" ]
then
  (cd ..; docker build . -t "${CHAIN_IMAGE}:${CHAIN_A_TAG}")
fi

go test -v ./tests/... --run ${ENTRY_POINT} -testify.m ^${TEST}$
