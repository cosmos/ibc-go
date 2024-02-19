#!/bin/bash

set -eou pipefail

# build_wasm_image extracts the correct libwasm version and checksum
# based on the go.mod and builds a docker image with the provided tag.
function build_wasm_image(){
  local version="$(scripts/get-libwasm-version.py --get-version)"
  local checksum="$(scripts/get-libwasm-version.py --get-checksum)"
  docker build . -t "${1}" -f modules/light-clients/08-wasm/Dockerfile --build-arg LIBWASM_VERSION=${version} --build-arg LIBWASM_CHECKSUM=${checksum}
}

# default to latest if no tag is specified.
TAG="${1:-ibc-go-wasm-simd:latest}"

build_wasm_image "${TAG}"

# if [ -z "$SIMD_BIN" ]; then echo "SIMD_BIN is not set. Make sure to run make install before"; exit 1; fi
# echo "using $SIMD_BIN"
# if [ -d "$($SIMD_BIN config home)" ]; then rm -r $($SIMD_BIN config home); fi
simd config set client chain-id simapp-1
simd config set client keyring-backend test
simd config set app api.enable true
simd keys add alice --keyring-backend test 
simd keys add bob --keyring-backend test 
simd init test --chain-id simapp-1 
simd genesis add-genesis-account alice 5000000000stake --keyring-backend test
simd genesis add-genesis-account bob 5000000000stake --keyring-backend test
simd genesis gentx alice 1000000stake --chain-id simapp-1
simd genesis collect-gentxs

BYTES=$(<./modules/light-clients/08-wasm/contracts/ics07_tendermint_cw.wasm.gz)

jq --arg contract "$BYTES" '.wasm["contracts"]=[{"code_bytes": $contract}]' ~/.simapp/config/genesis.json > temp.json && mv temp.json ~/.simapp/config/genesis.json