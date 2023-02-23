#!/usr/bin/env bash

set -euo pipefail

# lint_modified_go_files runs the linter only if changes have been made to any go files.
function lint_modified_go_files() {
  local go_files="$(git diff --name-only | grep \.go$)"
  for f in $go_files; do
    local dir_name="$(dirname $f)"
    golangci-lint run "${dir_name}" --fix --out-format=tab --issues-exit-code=0
  done
}

lint_modified_go_files
