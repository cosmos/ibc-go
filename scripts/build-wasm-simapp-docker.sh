#!/bin/bash

set -eou pipefail

# build_wasm_image extracts the correct libwasm version and checksum
# based on the go.mod and builds a docker image with the provided tag.
function build_wasm_image() {
  local version="$(scripts/get-libwasm-version.py --get-version)"
  local checksum="$(scripts/get-libwasm-version.py --get-checksum)"
  docker build . -t "${1}" -f modules/light-clients/08-wasm/Dockerfile --build-arg LIBWASM_VERSION=${version} --build-arg LIBWASM_CHECKSUM=${checksum}
}

# default to latest if no tag is specified.
TAG="${1:-ibc-go-wasm-simd:latest}"

build_wasm_image "${TAG}"
