#!/bin/bash

set -euo pipefail

ENTRY_POINT="${1}"
TEST="${2}"

export CHAIN_A_TAG="${CHAIN_A_TAG:-main}"
export CHAIN_IMAGE="${CHAIN_IMAGE:-ghcr.io/cosmos/ibc-go-simd}"
export CHAIN_BINARY="${CHAIN_BINARY:-simd}"

go test -v ./tests/... --run ${ENTRY_POINT} -testify.m ^${TEST}$
