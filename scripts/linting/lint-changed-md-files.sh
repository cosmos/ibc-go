#!/usr/bin/env bash

set -euo pipefail

# lint_modified_markdown_files runs the linter only if changes have been made to any md files.
function lint_modified_markdown_files() {
  local markdown_files="$(git diff --name-only | grep \.md$)"
  for f in $markdown_files; do
    markdownlint "${f}" --fix
  done
}

lint_modified_markdown_files
