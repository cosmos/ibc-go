#!/usr/bin/env bash

# Constants
WASM_IMPORT="github.com/CosmWasm/wasmvm/v2"

# Functions
function show_help {
  echo "Wasm dockerfile utility

Usage: $0 [--get-version | --get-checksum] [--wasm-library <library>] [--wasm-go-mod-path <path>]

Options:
  --get-version            Get the current version of CosmWasm specified in wasm module.
  --get-checksum           Returns the checksum of the libwasm library for the provided version.
  --wasm-library <library> The name of the library to return the checksum for (default: libwasmvm_muslc.x86_64.a).
  --wasm-go-mod-path <path> The relative path to the go.mod file for the wasm module (default: modules/light-clients/08-wasm/go.mod).
"
}

function get_wasm_version {
  local wasm_go_mod_path="$1"
  if [[ ! -f "$wasm_go_mod_path" ]]; then
    echo "Error: go.mod file not found at $wasm_go_mod_path"
    exit 1
  fi

  grep "$WASM_IMPORT" "$wasm_go_mod_path" | awk '{print $2}' || {
    echo "Error: Could not find $WASM_IMPORT in $wasm_go_mod_path"
    exit 1
  }
}

function get_wasm_lib_checksum {
  local wasm_version="$1"
  local wasm_lib="$2"
  local checksums_url="https://github.com/CosmWasm/wasmvm/releases/download/${wasm_version}/checksums.txt"

  local checksum
  checksum=$(curl -sSL "$checksums_url" | grep "$wasm_lib" | awk '{print $1}') || {
    echo "Error: Could not retrieve checksum for $wasm_lib from $checksums_url"
    exit 1
  }

  if [[ -z "$checksum" ]]; then
    echo "Error: Could not find $wasm_lib in $checksums_url"
    exit 1
  fi

  echo "$checksum"
}

# Default values
wasm_library="libwasmvm_muslc.x86_64.a"
wasm_go_mod_path="modules/light-clients/08-wasm/go.mod"

# Parse arguments
get_version=false
get_checksum=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --get-version)
      get_version=true
      shift
      ;;
    --get-checksum)
      get_checksum=true
      shift
      ;;
    --wasm-library)
      wasm_library="$2"
      shift 2
      ;;
    --wasm-go-mod-path)
      wasm_go_mod_path="$2"
      shift 2
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      echo "Error: Unknown argument $1"
      show_help
      exit 1
      ;;
  esac
done

# Main logic
if $get_version; then
  get_wasm_version "$wasm_go_mod_path"
  exit 0
fi

if $get_checksum; then
  wasm_version=$(get_wasm_version "$wasm_go_mod_path")
  get_wasm_lib_checksum "$wasm_version" "$wasm_library"
  exit 0
fi

echo "Error: No action specified. Use --get-version or --get-checksum."
show_help
exit 1
