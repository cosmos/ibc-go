#!/usr/bin/env bash

set -e -o pipefail

REPO_ROOT="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )/.." &> /dev/null && pwd )"
export REPO_ROOT

lint_module() {
  local root="$1"
  local dirname="$(dirname "$root")"
  echo "Linting $1"
  shift
  set -x
  cd $dirname &&
    golangci-lint run ./... -c "${REPO_ROOT}/.golangci.yml" "$@"
  set +x

}
export -f lint_module

find "${REPO_ROOT}" -type f -name go.mod -print0 |
  xargs -0 -I{} bash -c 'lint_module "$@"' _ {} "$@"
