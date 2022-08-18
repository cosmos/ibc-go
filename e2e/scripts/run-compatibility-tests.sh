#!/bin/bash

set -Eeou pipefail

VERSION="${1:-main}"

echo "## Test Results"
echo ""
jq -c -r '.[]' "scripts/test-matricies/${VERSION}/test-matrix.json" | while read arguments; do
    test_name="$(echo ${arguments} | jq -r -c '."test-entry-point"')"
    chain_a_tag="$(echo ${arguments} | jq -r -c '."chain-a-tag"')"
    chain_b_tag="$(echo ${arguments} | jq -r -c '."chain-b-tag"')"

    # manually trigger a workflow using the entry from the list
    echo ${arguments} | gh workflow run e2e-manual-simd.yaml --json > /dev/null
    # wait some time for the workflow to be started
    sleep 2

    # extract the id of the workflow we just started
    run_id="$(gh run list --workflow=e2e-manual-simd.yaml | grep workflow_dispatch | grep -Eo "[0-9]{9,11}" | head -n 1)"

    echo "Test ${test_name}"
    echo "Chain A: ${chain_a_tag}"
    echo "Chain B: ${chain_b_tag}"
    echo "Workflow Run https://github.com/cosmos/ibc-go/actions/runs/${run_id}"
    echo ""
done
