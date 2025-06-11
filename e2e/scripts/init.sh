#!/bin/bash

set -euo pipefail

# TODO: when we are using config files for CI we can remove this.
# ref: https://github.com/cosmos/ibc-go/issues/4697
# if running in CI we just use env vars.
if [[ "${CI:-}" = "true" ]]; then
  exit 0
fi

# ensure_config_file makes sure there is a config file for the e2e tests either by creating a new one using the sample,
# it is copied to either the default location or the specified env location.
function ensure_config_file(){
  local config_file_path="${HOME}/.ibc-go-e2e-config.yaml"
  if [[ ! -z "${E2E_CONFIG_PATH:-}" ]]; then
    config_file_path="${E2E_CONFIG_PATH}"
  fi
  if [[ ! -f "${config_file_path}" ]]; then
    echo "creating e2e config file from sample."
    echo "copying sample.config.yaml to ${config_file_path}"
    cp sample.config.yaml "${config_file_path}"
  fi
  echo "using config file at ${config_file_path} for e2e test"
}

ensure_config_file
