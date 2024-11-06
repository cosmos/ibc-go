# Testing Design

> This document outlines the testing CI structure as of Nov 6, 2024

## Unit / Integration tests

The unit and integration tests are run on every PR, they perform basic unit tests as well as integration tests.
The integration tests typically consist of setting up 2 in memory chains and test things like RPC endpoints and keeper
calls. Some functionality is mocked such as utility functions that commit blocks and modify time. Relaying is done by
the tests themselves by parsing events from tx responses.

These tests can be found in a `testing` directory for each of the various simapps that exist. These different simapps
are wired up with different features.

> Note: in the future, we would like to remove having multiple simapps, as this adds a lot of overhead to the testing.

In order to run unit / integration tests locally, you can run `make test` from the root of the repo.

In CI, `make test` is not called, but `go test` is called directly. The workflow for this can be found [here](../.github/workflows/test.yml)

# E2E tests

The main goal of the E2E tests, is to ensure that all components of the stack work as expected, with multiple chains
deployed and an actual relayer.

E2E tests are all located [here](./tests)



- workflow call
- tests run on main
- Go script
- workflow call
- build images on prs

## Manual E2E Tests

- mostly legacy workflows that don't get run much anymore, might be safe to remove these in the future.
- 

## Multiple simapps

- improvement, remove simapps

# E2E Fork tests

- build image locally, prevent creds from being passed

# Upgrade Tests

- mostly run manually

# Compatibility Tests

- json files
- run permutations of versions

# improvements
- enable full parallelization of tests
- improvement, remove simapps
- not need to pre-create relayers based
- script to dynamically generate compatibility matricies based on annotations.
- make tests more composable (less duplication, potentially remove the need for version checking in tests)