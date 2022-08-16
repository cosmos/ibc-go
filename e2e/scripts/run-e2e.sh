#!/bin/bash

set -euo pipefail

SUITE="${1}"
TEST="${2}"
CHAIN_BINARY="${3:-simd}"

export CHAIN_A_TAG="${CHAIN_A_TAG:-latest}"
export SIMD_IMAGE="${CHAIN_A_IMAGE:-ibc-go-simd}"

# In CI, the docker images will be built separately.
# context for building the image is one directory up.
if [ "${CI:-}" != "true" ]
then
  (cd ..; docker build . -t "${SIMD_IMAGE}:${CHAIN_A_TAG}")
fi

go test -v ./ --run ${SUITE} -testify.m ^${TEST}$ --args ${CHAIN_BINARY}
