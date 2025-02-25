#!/bin/bash

set -eou pipefail  # Exit on error, undefined variable, or pipe failure

# Enable debugging if VERBOSE=1
[[ "${VERBOSE:-0}" == "1" ]] && set -x

# Ensure the script is being run from the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR/.."
DOCKERFILE_PATH="modules/light-clients/08-wasm/Dockerfile"
GO_MOD_PATH="$PROJECT_ROOT/modules/light-clients/08-wasm/go.mod"

# Ensure required commands are available
for cmd in curl grep awk docker; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "ERROR: Required command '$cmd' not found. Please install it first."
    exit 1
  fi
done

# Ensure required files exist
if [[ ! -f "$PROJECT_ROOT/$DOCKERFILE_PATH" ]]; then
  echo "ERROR: Dockerfile not found at $DOCKERFILE_PATH"
  exit 1
fi

if [[ ! -f "$GO_MOD_PATH" ]]; then
  echo "ERROR: go.mod file not found at $GO_MOD_PATH"
  exit 1
fi

# Extract WASM version and checksum manually
WASM_VERSION=$(grep "github.com/CosmWasm/wasmvm/v2" "$GO_MOD_PATH" | awk '{print $2}')
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

# Function to build the Docker image
build_wasm_image() {
  local tag="${1}"
  echo "Building Docker image with tag: $tag"
  
  docker build . -t "$tag" \
    -f "$DOCKERFILE_PATH" \
    --build-arg LIBWASM_VERSION="$WASM_VERSION" \
    --build-arg LIBWASM_CHECKSUM="$WASM_CHECKSUM"
}

# default to latest if no tag is specified.
TAG="${1:-ibc-go-wasm-simd:latest}"

build_wasm_image "${TAG}"