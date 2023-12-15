#!/bin/bash

set -eo pipefail

ENTRY_POINT="${1}"
export KEEP_CONTAINTERS='true'

function _check_env() {
      if [ "$ENTRY_POINT" = "" ]; then
          echo "requires an entrypoint"
          exit 1
      fi
}

function _verify_jq() {
      if ! command -v jq > /dev/null ; then
          echo "jq is required to extract test entrypoint."
          exit 1
      fi
}

function _verify_dependencies() {
    _check_env
    # jq is always required to determine the entrypoint of the test.
    _verify_jq
}

_verify_dependencies

# find the name of the file that has this test in it.
test_file="$(grep --recursive --files-with-matches './' -e "${ENTRY_POINT}(t")"

# we run the test on the directory as specific files may reference types in other files but within the package.
test_dir="$(dirname $test_file)"

# run the test file directly, this allows log output to be streamed directly in the terminal sessions
# without needed to wait for the test to finish.
# it shouldn't take 30m, but the wasm test can be quite slow, so we can be generous.
go test -v "${test_dir}" --run ${ENTRY_POINT} -timeout 150m