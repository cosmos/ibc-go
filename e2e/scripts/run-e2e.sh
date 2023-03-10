#!/bin/bash

set -eo pipefail

TEST="${1}"
ENTRY_POINT="${2:-}"

export CHAIN_A_TAG="${CHAIN_A_TAG:-latest}"
export CHAIN_IMAGE="${CHAIN_IMAGE:-ibc-go-simd}"
export CHAIN_BINARY="${CHAIN_BINARY:-simd}"

# if jq is installed, we can automatically determine the test entrypoint.
if command -v jq; then
   cd ..
   ENTRY_POINT="$(go run -mod=readonly cmd/build_test_matrix/main.go | jq -r --arg TEST "${TEST}" '.include[] | select( .test == $TEST)  | .entrypoint')"
   cd e2e
fi

echo $ENTRY_POINT

go test -v ./tests/... --run ${ENTRY_POINT} -testify.m ^${TEST}$
