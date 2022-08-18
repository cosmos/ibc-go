#!/bin/bash

set -Eeou pipefail

VERSION="${1:-main}"

echo "### Backwards compatibility tests"
echo "## Version: ${VERSION}"

jq -c -r '.[]' "scripts/test-matricies/${VERSION}/test-matrix.json" | while read arguments; do
    test_name="$(echo ${arguments} | jq -r -c '."test-entry-point"')"
    test_arguments="$(echo ${arguments} | jq -r -c '."tests"')"
    echo ${test_arguments} | jq -c -r '.[]' | while read test; do
        chain_a_tag="$(echo ${test} | jq -r -c '."chain-a-tag"')"
        chain_b_tag="$(echo ${test} | jq -r -c '."chain-b-tag"')"

        # manually trigger a workflow using each entry from the list
        gh workflow run e2e-manual-simd.yaml -f chain-a-tag="${chain_a_tag}" -f chain-b-tag="${chain_b_tag}" -f test-entry-point="${test_name}" > /dev/null
        # it takes some time for the test to appear in the list, we need to wait for it to show up.
        sleep 2
        # this assumes nobody else has run a manual workflow in the last 2 seconds
        run_id="$(gh run list --workflow=e2e-manual-simd.yaml | grep workflow_dispatch | grep -Eo "[0-9]{9,11}" | head -n 1)"
        echo "[${test_name} chain A (${chain_a_tag}) -> chain B (${chain_b_tag})](https://github.com/cosmos/ibc-go/actions/runs/${run_id})"
        echo ""
    done
done
