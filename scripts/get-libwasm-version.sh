#!/bin/bash

# Default values
WASM_GO_MOD_PATH="modules/light-clients/08-wasm/go.mod"
WASM_LIBRARY="libwasmvm_muslc.x86_64.a"
WASM_IMPORT="github.com/CosmWasm/wasmvm/v2"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --get-version)
      GET_VERSION=true
      shift
      ;;
    --get-checksum)
      GET_CHECKSUM=true
      shift
      ;;
    --wasm-library)
      WASM_LIBRARY="$2"
      shift 2
      ;;
    --wasm-go-mod-path)
      WASM_GO_MOD_PATH="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Function to get the wasm version from go.mod
get_wasm_version() {
  if [ ! -f "$WASM_GO_MOD_PATH" ]; then
    echo "Error: $WASM_GO_MOD_PATH not found" >&2
    exit 1
  fi
  
  VERSION=$(grep "$WASM_IMPORT" "$WASM_GO_MOD_PATH" | awk '{print $2}')
  
  if [ -z "$VERSION" ]; then
    echo "Error: Could not find $WASM_IMPORT in $WASM_GO_MOD_PATH" >&2
    exit 1
  fi
  
  echo "$VERSION"
}

# Function to get the checksum for a specific library
get_wasm_lib_checksum() {
  local VERSION=$1
  local LIBRARY=$2
  local CHECKSUMS_URL="https://github.com/CosmWasm/wasmvm/releases/download/${VERSION}/checksums.txt"
  
  CHECKSUMS=$(curl -s -f "$CHECKSUMS_URL")
  
  if [ $? -ne 0 ]; then
    echo "Error: Failed to fetch checksums from $CHECKSUMS_URL" >&2
    exit 1
  fi
  
  CHECKSUM=$(echo "$CHECKSUMS" | grep "$LIBRARY" | awk '{print $1}')
  
  if [ -z "$CHECKSUM" ]; then
    echo "Error: Could not find $LIBRARY in checksums" >&2
    exit 1
  fi
  
  echo "$CHECKSUM"
}

# Main logic
if [ "$GET_VERSION" = true ]; then
  get_wasm_version
elif [ "$GET_CHECKSUM" = true ]; then
  VERSION=$(get_wasm_version)
  get_wasm_lib_checksum "$VERSION" "$WASM_LIBRARY"
else
  echo "Error: Must specify either --get-version or --get-checksum"
  exit 1
fi
