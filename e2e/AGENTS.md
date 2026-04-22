# AGENTS.md

This subtree is a separate Go module for containerized end-to-end tests.

## Look here first
- `Makefile`
- `README.md`
- `scripts/init.sh`
- `scripts/run-e2e.sh`
- `testsuite/`
- `../cmd/build_test_matrix/main.go`

## Run and validate
Run from `e2e/`:
- `make init`
- `make e2e-test test=<TestName>`
- `make e2e-suite entrypoint=<SuiteEntryPoint>`
- `go run ../cmd/build_test_matrix/main.go | jq`

## Constraints
- Docker is required.
- `jq` is required by `scripts/run-e2e.sh`; `fzf` is only needed for interactive selection when `test` or `entrypoint` is omitted.
- Config comes from `~/.ibc-go-e2e-config.yaml` or `E2E_CONFIG_PATH`; `make init` copies `sample.config.yaml` when missing.
- Keep the `//go:build !test_e2e` constraint on suite files so repo-root `make test` does not compile the containerized suites.
- Keep one suite entrypoint per file; test-matrix generation assumes all tests for a suite live in the same file.
