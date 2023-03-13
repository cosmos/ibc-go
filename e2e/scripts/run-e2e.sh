#!/bin/bash

set -euo pipefail

ENTRY_POINT="${1}"
TEST="${2}"

go test -v ./tests/... --run ${ENTRY_POINT} -testify.m ^${TEST}$
