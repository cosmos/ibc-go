#!/bin/bash

# check_golangci_lint_version ensures that the local version of golangci-lint
# matches the version specified in .github/workflows/golangci.yml. This makes sure
# that if the pre-commit hook is run locally, the changes should align with CI.
function check_golangci_lint_version(){
  local git_root="$(git rev-parse --show-toplevel)"

  # extract the version of golangci-lint from the CI workflow file.
  local golang_lint_ci_version="$(grep ' version' ${git_root}/.github/workflows/golangci.yml | awk '{ print $NF }')"

  # sample output of "golangci-lint --version"
  # golangci-lint has version 1.53.2 built with go1.20.4 from 59a7aaf on 2023-06-03T10:44:21Z
  local local_golang_lint_version="$(golangci-lint --version | awk '{ print $4 }')"

  if [[ "${golang_lint_ci_version}" != "${local_golang_lint_version}" ]]; then
    echo "local golangci-lint (${local_golang_lint_version}) must be upgraded to ${golang_lint_ci_version}"
    echo "aborting pre-commit hook"
    exit 1
  fi
}

# run_hook formats all modified go files and adds them to the commit.
function run_hook() {
  make lint-fix-changed
  echo "formatting any changed go files"
  go_files="$(git diff --name-only | grep \.go$)"
  for f in $go_files; do
    git add $f
  done
}

check_golangci_lint_version
run_hook
