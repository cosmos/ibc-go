#!/bin/bash

set -Eeou pipefail

VERSION="${1}"
LATEST_TAG="$(git tag -l | grep "v${VERSION}" | sort -V | tail -n 1)"
CURRENT_BRANCH="$(git branch --show-current)"
echo $LATEST_TAG

jq -c -r '.[]' scripts/test-matrix.json | while read i; do
    # manually trigger a workflow using the entry from the list
    echo ${i} | gh workflow run e2e-manual-simd.yaml --json
    # wait some time for the workflow to be started
    echo "waiting for task to start..."
    sleep 2
    # extract the id of the workflow we just started
    run_id="$(gh run list --workflow=e2e-manual-simd.yaml | grep workflow_dispatch | grep -Eo "[0-9]{9,11}" | head -n 1)"
    # open the workflow in a browser
    gh run view "${run_id}" --web
done
