#!/bin/bash

set -Eeou pipefail

function run_full_compatibility_suite(){
    local release_branch="${1}"
    gh workflow run e2e-compatibility.yaml -f release-branch=${release_branch}
    sleep 5 # can take some time for the workflow to appear
    run_id="$(gh run list "--workflow=e2e-compatibility.yaml" | grep workflow_dispatch | grep -Eo "[0-9]{9,11}" | head -n 1)"
    gh run view ${run_id} --web
}

RELEASE_BRANCH="${1}"
run_full_compatibility_suite "${RELEASE_BRANCH}"
