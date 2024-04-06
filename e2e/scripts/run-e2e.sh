#!/bin/bash

set -eo pipefail

TEST="${1}"
ENTRY_POINT="${2:-}"

function _verify_jq() {
      if ! command -v jq > /dev/null ; then
          echo "jq is required to extract test entrypoint."
          exit 1
      fi
}

function _verify_fzf() {
      if ! command -v fzf > /dev/null ; then
          echo "fzf is required to interactively select a test."
          exit 1
      fi
}

function _verify_dependencies() {
    if [ -z "${TEST}" ]; then
        # fzf is only required if we are not explicitly specifying a test.
        _verify_fzf
    fi
    # jq is always required to determine the entrypoint of the test.
    _verify_jq
}


# _select_test_config lets you dynamically select a test config for the specific test.
function _select_test_config() {
      # if an argument is provided, it is used directly. This enables the drop down selection with fzf.
      if [ -n "${1:-}" ]; then
          echo "$1"
          return
      else
          ls -1 dev-configs | fzf
      fi
}

# _get_test returns the test that should be used in the e2e test. If an argument is provided, that argument
# is returned. Otherwise, fzf is used to interactively choose from all available tests.
function _get_test(){
    # if an argument is provided, it is used directly. This enables the drop down selection with fzf.
    if [ -n "${1:-}" ]; then
        echo "$1"
        return
    # if fzf and jq are installed, we can use them to provide an interactive mechanism to select from all available tests.
    else
        cd ..
        go run -mod=readonly cmd/build_test_matrix/main.go | jq  -r '.include[] | .test' | fzf
        cd - > /dev/null
    fi
}

_verify_dependencies

# if the dev configs directory is present, enable fzf completion to select a test config file to use.
if [[ -d "dev-configs"  ]]; then
  export E2E_CONFIG_PATH="$(pwd)/dev-configs/$(_select_test_config)"
  echo "Using configuration file at ${E2E_CONFIG_PATH}"
fi

# if test is set, that is used directly, otherwise the test can be interactively provided if fzf is installed.
TEST="$(_get_test ${TEST})"

# if jq is installed, we can automatically determine the test entrypoint.
if command -v jq > /dev/null; then
   cd ..
   ENTRY_POINT="$(go run -mod=readonly cmd/build_test_matrix/main.go | jq -r --arg TEST "${TEST}" '.include[] | select( .test == $TEST)  | .entrypoint')"
   cd - > /dev/null
fi


# find the name of the file that has this test in it.
test_file="$(grep --recursive --files-with-matches './' -e "${TEST}()")"

# we run the test on the directory as specific files may reference types in other files but within the package.
test_dir="$(dirname $test_file)"

# run the test file directly, this allows log output to be streamed directly in the terminal sessions
# without needed to wait for the test to finish.
# it shouldn't take 30m, but the wasm test can be quite slow, so we can be generous.
go test -v "${test_dir}" --run ${ENTRY_POINT} -testify.m ^${TEST}$ -timeout 30m
