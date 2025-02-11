#!/bin/bash

set -eou pipefail

# Ensure the script is being run from the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR/.."
DOCKERFILE_PATH="modules/light-clients/08-wasm/Dockerfile"

# Ensure required files exist
if [[ ! -f "$PROJECT_ROOT/$DOCKERFILE_PATH" ]]; then
  echo "ERROR: Dockerfile not found at $DOCKERFILE_PATH"
  exit 1
fi

if [[ ! -f "$PROJECT_ROOT/modules/light-clients/08-wasm/go.mod" ]]; then
  echo "ERROR: go.mod file not found!"
  exit 1
fi

# Extract WASM version and checksum manually
WASM_VERSION=$(grep "github.com/CosmWasm/wasmvm/v2" "$PROJECT_ROOT/modules/light-clients/08-wasm/go.mod" | awk '{print $2}')
if [[ -z "$WASM_VERSION" ]]; then
  echo "ERROR: Failed to extract WASM version from go.mod"
  exit 1
fi

WASM_CHECKSUM=$(curl -sL "https://github.com/CosmWasm/wasmvm/releases/download/${WASM_VERSION}/checksums.txt" | grep "libwasmvm_muslc.x86_64.a" | awk '{print $1}')
if [[ -z "$WASM_CHECKSUM" ]]; then
  echo "ERROR: Failed to extract checksum from WASM repository"
  exit 1
fi

echo "Using WASM_VERSION=${WASM_VERSION}"
echo "Using WASM_CHECKSUM=${WASM_CHECKSUM}"

# Build the Docker image
function build_wasm_image(){
  docker build . -t "${1}" \
    -f "$DOCKERFILE_PATH" \
    --build-arg LIBWASM_VERSION="${WASM_VERSION}" \
    --build-arg LIBWASM_CHECKSUM="${WASM_CHECKSUM}"
}

# default to latest if no tag is specified.
TAG="${1:-ibc-go-wasm-simd:latest}"

build_wasm_image "${TAG}"