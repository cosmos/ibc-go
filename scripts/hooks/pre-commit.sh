#!/bin/bash

# check_golangci_lint_version ensures that the local version of golangci-lint
# matches the version specified in .github/workflows/golangci.yml. This makes sure
# that if the pre-commit hook is run locally, the changes should align with CI.
function check_golangci_lint_version(){
  local git_root="$(git rev-parse --show-toplevel)"

  # Note: we are explicitly stripping out the 'v' prefix from the versions. Different versions of
  # golangci-lint have different version formats. For example, v1.27.0 (if installed with go get) vs 1.27.0 (installed with curl).

  # extract the version of golangci-lint from the CI workflow file.
  local workflow_golangci_lint_version="$(grep ' version' ${git_root}/.github/workflows/golangci.yml | awk '{ print $NF }' | sed "s/v//g" )"

  local local_golangci_lint_version="$(golangci-lint version --format short | grep '[0-9\.]'| sed "s/v//g")"

  if [[ "${workflow_golangci_lint_version}" != "${local_golangci_lint_version}" ]]; then
    echo "local golangci-lint (${local_golangci_lint_version}) must be upgraded to ${workflow_golangci_lint_version}"
    echo "aborting pre-commit hook"
    exit 1
  fi
}

function lint_and_add_modified_go_files() {
  local go_file_dirs="$(git diff --name-only --diff-filter=d | grep \.go$ | grep -v \.pb\.go$ | xargs dirname | sort | uniq)"
  for dir_name in $go_file_dirs; do
    golangci-lint run "${dir_name}" --fix --out-format=tab --issues-exit-code=0
    echo "adding ${dir_name} to git index"
    git add $dir_name
  done
}

function run_proto_all_if_needed() {
  local before_files=$(git status --porcelain | awk '{print $2}')

  # Run make proto-all
  make proto-all

  local after_files=$(git status --porcelain | awk '{print $2}')
  local changed_files=$(comm -13 <(echo "$before_files" | sort) <(echo "$after_files" | sort))

  if [[ -n "$changed_files" ]]; then
    echo "The following files have been modified by 'make proto-all' and have been added to the git index:"
    for file in $changed_files; do
      echo "$file"
      git add "$file"
    done

    # Add the modified .proto files as well
    local modified_proto_files=$(echo "$before_files" "$after_files" | tr ' ' '\n' | grep '\.proto$' | sort | uniq)
    if [[ -n "$modified_proto_files" ]]; then
      echo "The following .proto files have been modified and have been added to the git index:"
      for proto_file in $modified_proto_files; do
        echo "$proto_file"
        git add "$proto_file"
      done
    fi
  fi
}

check_golangci_lint_version
run_proto_all_if_needed
lint_and_add_modified_go_files
