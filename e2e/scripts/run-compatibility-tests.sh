#!/bin/bash

set -Eeou pipefail

# run_gh_workflow runs a single github workflow and outputs the workflow information in a markdown format.
function run_gh_workflow(){
        local test_entry_point="${1}"
        local chain_binary="${2}"
        local chain_a_tag="${3}"
        local chain_b_tag="${4}"

        # any changes to the workflows on this branch will be used.
        local current_branch="$(git branch --show-current)"

        # manually trigger a workflow using each entry from the list
        gh workflow run "e2e-manual-${chain_binary}.yaml" --ref="${current_branch}" \
          -f chain-a-tag="${chain_a_tag}" \
          -f chain-b-tag="${chain_b_tag}" \
          -f test-entry-point="${test_entry_point}" > /dev/null
        # it takes some time for the test to appear in the list, we need to wait for it to show up.
        sleep 2
        # this assumes nobody else has run a manual workflow in the last 2 seconds
        run_id="$(gh run list "--workflow=e2e-manual-${chain_binary}.yaml" | grep workflow_dispatch | grep -Eo "[0-9]{9,11}" | head -n 1)"
        echo "- [ ] [${test_entry_point} chain A (${chain_a_tag}) -> chain B (${chain_b_tag})](https://github.com/cosmos/ibc-go/actions/runs/${run_id})"
        echo ""
}

# run_full_compatibility_suite runs all tests specified in the test-matrix.json file.
function run_full_compatibility_suite(){
    local matrix_version="${1}"
    local matrix_file_path="${2:-"scripts/test-matricies/${matrix_version}/test-matrix.json"}"

    echo "### Backwards compatibility tests"
    echo "#### Matrix Version: ${matrix_version}"

    jq -c -r '.[]' "${matrix_file_path}" | while read arguments; do
        test_entry_point="$(echo ${arguments} | jq -r -c '."test-entry-point"')"
        test_arguments="$(echo ${arguments} | jq -r -c '."tests"')"
        chain_binary="$(echo ${arguments} | jq -r -c '."chain-binary"')"

        echo ${test_arguments} | jq -c -r '.[]' | while read test; do
            chain_a_tag="$(echo ${test} | jq -r -c '."chain-a-tag"')"
            chain_b_tag="$(echo ${test} | jq -r -c '."chain-b-tag"')"
            run_gh_workflow "${test_entry_point}" "${chain_binary}" "${chain_a_tag}" "${chain_b_tag}"
        done
    done
}

VERSION_MATRIX="${1:-main}"
run_full_compatibility_suite "${VERSION_MATRIX}"
