# Table of Contents

1. [Running Tests](#running-tests)
2. [Adding a new test](#adding-a-new-test)
3. [Test design](#test-design)
   - a. [interchaintest](#interchaintest)
   - b. [CI configuration](#ci-configuration)
4. [Building images](#building-and-pushing-images)
5. [Compatibility Tests](#compatibility-tests)
   - a. [Running Compatibility Tests](#running-compatibility-tests)
   - b. [How Compatibility Tests Work](#how-compatibility-tests-work)
6. [Troubleshooting](#troubleshooting)
7. [Importable Workflow](#importable-workflow)
8. [Future Improvements](#future-improvements)

# Running tests

Tests can be run using a Makefile target under the e2e directory. `e2e/Makefile`

The tests can be configured using a configuration file or environment variables.

See the [minimal example](./sample.config.yaml) or [extended example](./sample.config.extended.yaml) to get started. The default location the tests look is `~/.ibc-go-e2e-config.yaml`
But this can be specified directly using the `E2E_CONFIG_PATH` environment variable.

It is possible to maintain multiple configuration files for tests. This can be useful when wanting to run the tests
using different images, relayers etc.

By creating an `./e2e/dev-configs` directory, and placing any number of configurations there. You will be prompted to choose
which configuration to use when running tests.

> Note: this requires fzf to be installed to support the interactive selection of configuration files.

There are several environment variables that alter the behaviour of the make target which will override any
options specified in your config file. These are primarily used for CI and are not required for local development.

See the extended sample config to understand all the available fields and their purposes.

> Note: when running tests locally, **no images are pushed** to the `ghcr.io/cosmos/ibc-go-simd` registry.
> The images which are used only exist locally only.

These environment variables allow us to run tests with arbitrary versions (from branches or releases) of simd and the go / hermes relayer.

Every time changes are pushed to a branch or to `main`, a new `simd` image is built and
pushed [here](https://github.com/orgs/cosmos/packages?repo_name=ibc-go).

On PRs, E2E tests will only run once the PR is marked as ready for review. This is to prevent unnecessary test runs on PRs that are still in progress.

> If you need the E2E tests to run, you can either run them locally, or you can mark the PR as R4R and then convert it back to a draft PR.

## Adding a new test

All tests should go under the [e2e](https://github.com/cosmos/ibc-go/tree/main/e2e) directory. When adding a new test, either add a new test function
to an existing test suite ***in the same file***, or create a new test suite in a new file and add test functions there.
New test files should follow the convention of `module_name_test.go`.

After creating a new test file, be sure to add a build constraint that ensures this file will **not** be included in the package to be built when
running tests locally via `make test`. For an example of this, see any of the existing test files.

New test suites should be composed of `testsuite.E2ETestSuite`. This type has lots of useful helper functionality that will
be quite common in most tests.

Override the default `SetupSuite` function with the number of chains required for the suite. Example:

```go
// SetupSuite sets up chains for the current test suite
func (s *ConnectionTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil) // This suite requires at most two chains.
}
```

> Note: see [here](#how-tests-are-run) for details about these requirements.

### Example of running a single test

> NOTE: environment variables can be set to override one or more config file variables, but the config file can still
> be used to set defaults.

```sh 

make e2e-test entrypoint=TestInterchainAccountsTestSuite test=TestMsgSubmitTx_SuccessfulTransfer
```

If `jq` is installed, you only need to specify the `test`.

If `fzf` is also installed, you only need to run `make e2e-test` and you will be prompted with interactive test
selection.

```sh
make e2e-test test=TestMsgSubmitTx_SuccessfulTransfer
```

> Note: sometimes it can be useful to make changes to [interchaintest](https://github.com/strangelove-ventures/interchaintest)
> when running tests locally. In order to do this, add the following line to
> e2e/go.mod

`replace github.com/strangelove-ventures/interchaintest => ../../interchaintest`

Or point it to any local checkout you have.

### Example of running a full testsuite

> NOTE: not all tests may support full parallel runs due to possible chain wide modifications such as params / gov
> proposals / chain restarts. See [When to Use t.Parallel()](#when-to-use-tparallel) for more information.

```sh 
make e2e-suite entrypoint=TestTransferTestSuite
```

Similar to running a single test, if `jq` and `fzf` are installed you can run `make e2e-suite` and be prompted
to interactively select a test suite to run.

### Running tests outside the context of the Makefile

In order to run tests outside the context of the Makefile (e.g. from an IDE)

The default location for a config file will be `~/.ibc-go-e2e-config.yaml` but this can be overridden by setting the
`E2E_CONFIG_PATH` environment variable.

This should be set to the path of a valid config file you want to use, setting this env will depend on the IDE being used.

## Test design

### interchaintest

These E2E tests use the [interchaintest framework](https://github.com/strangelove-ventures/interchaintest). This framework creates chains and relayers in containers and allows for arbitrary commands to be executed in the chain containers,
as well as allowing us to broadcast arbitrary messages which are signed on behalf of a user created in the test.

### Test Suites

In order for tests to be run in parallel, we create the chains in `SetupSuite`, and each test is in charge of 
creating clients/connections/channels for itself.

This is explicitly not being done in `SetupTest` to enable maximum control and flexibility over the channel creation
params. e.g. some tests may not want a channel created initially, and may want more flexibility over the channel creation itself.

### When to use t.Parallel()

tests should **not** be run in parallel when:

- the test is modifying chain wide state such as modifying params via a gov proposal.
- the test needs to perform a chain restart.
- the test must make assertions which may not be deterministic due to other tests. (e.g. the TotalEscrowForDenom may be modified between tests)

### CI Configuration

There are two main github actions for standard e2e tests.

[e2e.yaml](https://github.com/cosmos/ibc-go/blob/main/.github/workflows/e2e.yaml) which runs when collaborators create branches.

In `e2e.yaml`, the `simd` image is built and pushed to [a registry](https://github.com/orgs/cosmos/packages?repo_name=ibc-go) and every test
that is run uses the image that was built.

In `e2e-fork.yaml`, images are not pushed to this registry, but instead remain local to the host runner.

## How Tests Are Run

The tests use the `matrix` feature of Github Actions. The matrix is
dynamically generated using [this tool](https://github.com/cosmos/ibc-go/blob/main/cmd/build_test_matrix/main.go).

> Note: there is currently a limitation that all tests belonging to a test suite must be in the same file.
> In order to support test functions spread in different files, we would either need to manually maintain a matrix
> or update the script to account for this. The script assumes there is a single test suite per test file to avoid an
> overly complex generation process.

Which looks under the `e2e` directory, and creates a task for each test suite function.

This tool can be run locally to see which tests will be run in CI.

```sh
go run cmd/build_test_matrix/main.go | jq
```

This string is used to generate a test matrix in the Github Action that runs the E2E tests.

All tests will be run on different hosts when running `make e2e-test` but `make e2e-suite` will run multiple tests
in parallel on a shared host.

In a CI environment variables are passed to the test runner to configure test parameters, while locally using
environment variables is supported, but it is often more convenient to use configuration files.

## Building and pushing images

If we ever need to manually build and push an image, we can do so with the [Build Simd Image](../.github/workflows/build-simd-image-from-tag.yml) GitHub workflow.

This can be triggered manually from the UI by navigating to

`Actions` -> `Build Simd Image` -> `Run Workflow`

And providing the git tag.

> There are similar workflows for other simapps in the repo.

## Compatibility Tests

### Running Compatibility Tests

To trigger the compatibility tests for a release branch, you can trigger these manually from the Github UI.

This will build an image from the tip of the release branch and run all tests specified in the corresponding
E2E test annotations.

Navigate to `Actions` -> `Compatibility E2E` -> `Run Workflow` -> `release/v8.0.x`

> Note: this will spawn a large number of runners, and should only be used when there is a release candidate and
> and so should not be run regularly. We can rely on the regular E2Es on PRs for the most part.

### How Compatibility Tests Work

The compatibility tests are executed in [this workflow](../.github/workflows/e2e-compatibility.yaml). This workflow
will build an image for a specified release candidate based on the release branch as an input. And run the corresponding
jobs which are maintained under the `.github/compatibility-test-matrices` directory.

> At the moment these are manually maintained, but in the future we may be able to generate these matrices dynamically. See the [future improvements](#future-improvements) section for more details.

See [this example](https://github.com/cosmos/ibc-go/actions/runs/11645461969) to what the output of a compatibility test run looks like.

## Troubleshooting

- On Mac, after running a lot of tests, it can happen that containers start failing. To fix this, you can try clearing existing containers and restarting the docker daemon.

  This generally manifests itself as relayer or simd containers timing out during setup stages of the test. This doesn't happen in CI.

  ```bash
  # delete all images
  docker system prune -af
  ```

  This issue doesn't seem to occur on other operating systems.

### Accessing Logs

- When a test fails in GitHub. The logs of the test will be uploaded (viewable in the summary page of the workflow). Note: There
  may be some discrepancy in the logs collected and the output of interchaintest. The containers may run for a some
  time after the logs are collected, resulting in the displayed logs to differ slightly.

## Importable Workflow

This repository contains an [importable workflow](../.github/workflows/e2e-test-workflow-call.yml) that can be used from any other repository to test two chains. The workflow
can be used to test both non-IBC chains, and also IBC-enabled chains.

### Prerequisites

- In order to run this workflow, a docker container is required with tags for the versions you want to test.

- If you are running an upgrade, Have an upgrade handler in the chain binary which is being upgraded to.

> It's worth noting that all github repositories come with a built-in docker registry that makes it convenient to build and push images to.

[This workflow](https://github.com/cosmos/ibc-go/blob/1da651e5e117872499e3558c2a92f887369ae262/.github/workflows/release.yml#L35-L61) can be used as a reference for how to build a docker image
whenever a git tag is pushed.

### How to import the workflow

You can refer to [this example](https://github.com/cosmos/ibc-go/blob/2933906d1ed25ae6dce7b7d93aa429dfa94c5a23/.github/workflows/e2e-upgrade.yaml#L9-L19) when including this workflow in your repo.

The referenced job will do the following:

- Create two chains using the image found at `ghcr.io/cosmos/ibc-go-simd:v4.3.0`.
- Perform IBC transfers verifying core functionality.
- Upgrade chain A to `ghcr.io/cosmos/ibc-go-simd:v5.1.0` by executing a governance proposal and using the plan name `normal upgrade`.
- Perform additional IBC transfers and verifies the upgrade and migrations ran successfully.

> Note: The plan name will always be specific to your chain. In this instance `normal upgrade` is referring to [this upgrade handler](https://github.com/cosmos/ibc-go/blob/e9bc0bac38e84e1380ec08552cae15821143a6b6/testing/simapp/app.go#L923)

### Workflow Options

For a full list of options that can be passed to the workflow, see the [workflow file](../.github/workflows/e2e-test-workflow-call.yml).

## Future Improvements

- We are transitioning to running a single test per host, to running a full test suite per host. This will allow us to
  run more tests in parallel and reduce the time it takes to run the full suite. (see `make e2e-test` and `make e2e-suite`). Eventually we would like to run all tests with `make e2e-suite`.
  However, it may not be possible run compatibility tests in this fashion, as there was a concurrency issue in older versions of the SDK that caused chains to panic. See [this issue](https://github.com/cosmos/cosmos-sdk/pull/21073) for more details.
  Because of this, it may be necessary to run compatibility tests in a single test per host fashion.
- When running `make e2e-suite` we currently pre-create relayers, as each test uses a different relayer.
  An improvement would be to ensure that only enough relayers are created to satisfy the tests that are being run. This may require some changes to interchaintest and/or ibc-go test code. See [this issue](https://github.com/cosmos/ibc-go/issues/7578) for more details.
- There is a lot of duplication in the e2e test code. It would be nice to refactor the tests themselves to be more composable. For example, lots of tests perform an ics20 transfer, it would be nice to have
  this logic abstracted away into a helper function and be able to call this function from multiple tests.
- We currently have some e2e workflows that may no longer be necessary, such as the [e2e manual simd](../.github/workflows/e2e-manual-simd.yaml). These manual dispatch workflows were utilized a lot in the past, but are not used as much anymore.  
  It may be safe to remove these in the future. See [this issue](https://github.com/cosmos/ibc-go/issues/7579) for more details.
- We currently are maintaining a lot of different simapps, all with different configurations and wiring in their app.gos. Ideally, we only need to maintain a single one. Some refactoring will be necessary to achieve this. See [this](https://github.com/cosmos/ibc-go/issues/4527) related issue.
- The current parallel tests establish a connection and run each test on a different channel. This will need to be refactored once the IBC v2 is shipped, however consideration will need to be made for backwards compatibility tests. Both flows may need to be supported.
- CI still relies on passing lots of environment variables, it should be possible to provide a default CI configuration file to be used, and override specifics such as chain image, relayer image, etc. with environment variables. Or template configuration files in some way. This is tracked [here](https://github.com/cosmos/ibc-go/issues/4697).
- Currently, there is a single denom used for all tests. Ideally, there is a dynamically generated denom used on a per test level. The fact that a single denom is shared means certain features may not be testable in parallel, for example [this](https://github.com/cosmos/ibc-go/blob/main/e2e/tests/transfer/base_test.go#L94-L101).
