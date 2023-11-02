# Table of Contents

1. [How to write tests](#how-to-write-tests)
   - a. [Adding a new test](#adding-a-new-test)
   - b. [Running the tests with custom images](#running-tests-with-custom-images)
   - b. [Code samples](#code-samples)
     - [Setup](#setup)
     - [Creating test users](#creating-test-users)
     - [Waiting](#waiting)
     - [Query wallet balances](#query-wallet-balances)
     - [Broadcasting messages](#broadcasting-messages)
     - [Starting the relayer](#starting-the-relayer)
     - [Arbitrary commands](#arbitrary-commands)
     - [IBC transfer](#ibc-transfer)
2. [Test design](#test-design)
   - a. [interchaintest](#interchaintest)
   - b. [CI configuration](#ci-configuration)
3. [Github Workflows](#github-workflows)
4. [Running Compatibility Tests](#running-compatibility-tests)
5. [Troubleshooting](#troubleshooting)
6. [Importable Workflow](#importable-workflow)

# How to write tests

## Adding a new test

All tests should go under the [e2e](https://github.com/cosmos/ibc-go/tree/main/e2e) directory. When adding a new test, either add a new test function
to an existing test suite ***in the same file***, or create a new test suite in a new file and add test functions there.
New test files should follow the convention of `module_name_test.go`.

After creating a new test file, be sure to add a build constraint that ensures this file will **not** be included in the package to be built when
running tests locally via `make test`. For an example of this, see any of the existing test files.

New test suites should be composed of `testsuite.E2ETestSuite`. This type has lots of useful helper functionality that will
be quite common in most tests.

> Note: see [here](#how-tests-are-run) for details about these requirements.

## Running tests with custom images

Tests can be run using a Makefile target under the e2e directory. `e2e/Makefile`

The tests can be configured using a configuration file or environment variables.

See [the example](./sample.config.yaml) to get started. The default location the tests look is `~/.ibc-go-e2e-config.yaml`
But this can be specified directly using the `E2E_CONFIG_PATH` environment variable.

There are several environment variables that alter the behaviour of the make target which will override any
options specified in your config file.

| Environment Variable | Description                               | Default Value |
|----------------------|-------------------------------------------|---------------|
| CHAIN_IMAGE          | The image that will be used for the chain | ibc-go-simd   |
| CHAIN_A_TAG          | The tag used for chain B                  | latest        |
| CHAIN_B_TAG          | The tag used for chain A                  | latest        |
| CHAIN_BINARY         | The binary used in the container          | simd          |
| RELAYER_TAG          | The tag used for the relayer              | main          |
| RELAYER_ID           | The type of relayer to use (rly/hermes)   | hermes        |

> Note: when running tests locally, **no images are pushed** to the `ghcr.io/cosmos/ibc-go-simd` registry.
The images which are used only exist on your machine.

These environment variables allow us to run tests with arbitrary verions (from branches or released) of simd
and the go relayer.

Every time changes are pushed to a branch or to `main`, a new `simd` image is built and pushed [here](https://github.com/cosmos/ibc-go/pkgs/container/ibc-go-simd).

### Example Command

```sh
export CHAIN_IMAGE="ghcr.io/cosmos/ibc-go-simd"
export CHAIN_A_TAG="main"
export CHAIN_BINARY="simd"

# We can also specify different values for the chains if needed.
# they will default to the same as chain a.
# export CHAIN_B_TAG="main"

export RELAYER_TAG="v2.0.0"
make e2e-test entrypoint=TestInterchainAccountsTestSuite test=TestMsgSubmitTx_SuccessfulTransfer
```

If `jq` is installed, you only need to specify the `test`.

If `fzf` is also installed, you only need to run `make e2e-test` and you will be prompted with interactive test selection.

```sh
make e2e-test test=TestMsgSubmitTx_SuccessfulTransfer
```

> Note: sometimes it can be useful to make changes to [ibctest](https://github.com/strangelove-ventures/interchaintest) when running tests locally. In order to do this, add the following line to
e2e/go.mod

`replace github.com/strangelove-ventures/interchaintest => ../ibctest`

Or point it to any local checkout you have.

### Running tests in CI

To run tests in CI, you can checkout the ibc-go repo and provide these environment variables
to the CI task.

[This repo](https://github.com/chatton/ibc-go-e2e-demo) contains an example of how to do this with Github Actions.

## Code samples

### Setup

Every standard test will start with this. This creates two chains and a relayer,
initializes relayer accounts on both chains, establishes a connection and a channel
between the chains.

Both chains have started, but the relayer is not yet started.

The relayer should be started as part of the test if required. See [Starting the Relayer](#starting-the-relayer)

```go
relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
chainA, chainB := s.GetChains()
```

### Creating test users

There are helper functions to easily create users on both chains.

```go
chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
```

### Waiting

We can wait for some number of blocks on the specified chains if required.

```go
chainA, chainB := s.GetChains()
err := test.WaitForBlocks(ctx, 1, chainA, chainB)
s.Require().NoError(err)
```

### Query wallet balances

We can fetch balances of wallets on specific chains.

```go
chainABalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
s.Require().NoError(err)
```

### Broadcasting messages

We can broadcast arbitrary messages which are signed on behalf of users created in the test.

This example shows a multi message transaction being broadcast on chainA and signed on behalf of chainAWallet.

```go
relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
chainA, chainB := s.GetChains()

chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

t.Run("broadcast multi message transaction", func(t *testing.T){
  payPacketFeeMsg := feetypes.NewMsgPayPacketFee(testFee, channelA.PortID, channelA.ChannelID, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix), nil)
  transferMsg := transfertypes.NewMsgTransfer(channelA.PortID, channelA.ChannelID, transferAmount, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix), chainBWallet.Bech32Address(chainB.Config().Bech32Prefix), clienttypes.NewHeight(1, 1000), 0)
  resp, err := s.BroadcastMessages(ctx, chainA, chainAWallet, payPacketFeeMsg, transferMsg)

  s.AssertValidTxResponse(resp)
  s.Require().NoError(err)
})
```

### Starting the relayer

The relayer can be started with the following.

```go
t.Run("start relayer", func(t *testing.T) {
  s.StartRelayer(relayer)
})
```

### Arbitrary commands

Arbitrary commands can be executed on a given chain.

> Note: these commands will be fully configured to run on the chain executed on (home directory, ports configured etc.)

However, it is preferable to [broadcast messages](#broadcasting-messages) or use a gRPC query if possible.

```go
stdout, stderr, err := chainA.Exec(ctx, []string{"tx", "..."}, nil)
```

### IBC transfer

It is possible to send an IBC transfer in two ways.

Use the ibctest `Chain` interface (this ultimately does a docker exec)

```go
t.Run("send IBC transfer", func(t *testing.T) {
  chainATx, err = chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName, walletAmount, nil)
  s.Require().NoError(err)
  s.Require().NoError(chainATx.Validate(), "chain-a ibc transfer tx is invalid")
})
```

Broadcast a `MsgTransfer`.

```go
t.Run("send IBC transfer", func(t *testing.T){
  transferMsg := transfertypes.NewMsgTransfer(channelA.PortID, channelA.ChannelID, transferAmount, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix), chainBWallet.Bech32Address(chainB.Config().Bech32Prefix), clienttypes.NewHeight(1, 1000), 0)
  resp, err := s.BroadcastMessages(ctx, chainA, chainAWallet, transferMsg)
  s.AssertValidTxResponse(resp)
  s.Require().NoError(err)
})
```

## Test design

### interchaintest

These E2E tests use the [interchaintest framework](https://github.com/strangelove-ventures/interchaintest). This framework creates chains and relayers in containers and allows for arbitrary commands to be executed in the chain containers,
as well as allowing us to broadcast arbitrary messages which are signed on behalf of a user created in the test.

### CI configuration

There are two main github actions for e2e tests.

[e2e.yaml](https://github.com/cosmos/ibc-go/blob/main/.github/workflows/e2e.yaml) which runs when collaborators create branches.

[e2e-fork.yaml](https://github.com/cosmos/ibc-go/blob/main/.github/workflows/e2e-fork.yml) which runs when forks are created.

In `e2e.yaml`, the `simd` image is built and pushed to [a registry](https://github.com/cosmos/ibc-go/pkgs/container/ibc-go-simd) and every test
that is run uses the image that was built.

In `e2e-fork.yaml`, images are not pushed to this registry, but instead remain local to the host runner.

## How tests are run

The tests use the `matrix` feature of Github Actions. The matrix is
dynamically generated using [this command](https://github.com/cosmos/ibc-go/blob/main/cmd/build_test_matrix/main.go).

> Note: there is currently a limitation that all tests belonging to a test suite must be in the same file.
In order to support test functions spread in different files, we would either need to manually maintain a matrix
or update the script to account for this. The script assumes there is a single test suite per test file to avoid an overly complex
generation process.

Which looks under the `e2e` directory, and creates a task for each test suite function.

### Example

```go
// e2e/file_one_test.go
package e2e

func TestFeeMiddlewareTestSuite(t *testing.T) {
  suite.Run(t, new(FeeMiddlewareTestSuite))
}

type FeeMiddlewareTestSuite struct {
  testsuite.E2ETestSuite
}

func (s *FeeMiddlewareTestSuite) TestA() {}
func (s *FeeMiddlewareTestSuite) TestB() {}
func (s *FeeMiddlewareTestSuite) TestC() {}

```

```go
// e2e/file_two_test.go
package e2e

func TestTransferTestSuite(t *testing.T) {
  suite.Run(t, new(TransferTestSuite))
}

type TransferTestSuite struct {
  testsuite.E2ETestSuite
}

func (s *TransferTestSuite) TestD() {}
func (s *TransferTestSuite) TestE() {}
func (s *TransferTestSuite) TestF() {}

```

In the above example, the following would be generated.

```json
{
  "include": [
    {
      "entrypoint": "TestFeeMiddlewareTestSuite",
      "test": "TestA"
    },
    {
      "entrypoint": "TestFeeMiddlewareTestSuite",
      "test": "TestB"
    },
    {
      "entrypoint": "TestFeeMiddlewareTestSuite",
      "test": "TestC"
    },
    {
      "entrypoint": "TestTransferTestSuite",
      "test": "TestD"
    },
    {
      "entrypoint": "TestTransferTestSuite",
      "test": "TestE"
    },
    {
      "entrypoint": "TestTransferTestSuite",
      "test": "TestF"
    }
  ]
}
```

This string is used to generate a test matrix in the Github Action that runs the E2E tests.

All tests will be run on different hosts.

### Misceleneous

## GitHub Workflows

### Building and pushing a `simd` image

If we ever need to manually build and push an image, we can do so with the [Build Simd Image](../.github/workflows/build-simd-image-from-tag.yml) GitHub workflow.

This can be triggered manually from the UI by navigating to

`Actions` -> `Build Simd Image` -> `Run Workflow`

And providing the git tag.

Alternatively, the [gh](https://cli.github.com/) CLI tool can be used to trigger this workflow.

```bash
gh workflow run "Build Simd Image" -f tag=v3.0.0
```

## Running Compatibility Tests

To trigger the compatibility tests for a release branch, you can use the following command.

```bash
make compatibility-tests release_branch=release/v5.0.x
```

This will build an image from the tip of the release branch and run all tests specified in the corresponding
json matrix files under .github/compatibility-test-matrices and is equivalent to going to the Github UI and navigating to

`Actions` -> `Compatibility E2E` -> `Run Workflow` -> `release/v5.0.x`

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
  may be some discrepancy in the logs collected and the output of interchain test. The containers may run for a some
  time after the logs are collected, resulting in the displayed logs to differ slightly.

## Importable Workflow

This repository contains an [importable workflow](https://github.com/cosmos/ibc-go/blob/185a220244663457372185992cfc85ed9e458bf1/.github/workflows/e2e-compatibility-workflow-call.yaml) that can be used from any other repository to test chain upgrades. The workflow
can be used to test both non-IBC chains, and also IBC-enabled chains.

### Prerequisites

- In order to run this workflow, a docker container is required with tags for the versions you want to test.

- Have an upgrade handler in the chain binary which is being upgraded to.

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

| Workflow Field    | Purpose                                           |
|-------------------|---------------------------------------------------|
| chain-image       | The docker image to use for the test              |
| chain-a-tag       | The tag of chain A to use                         |
| chain-b-tag       | The tag of chain B to use                         |
| chain-upgrade-tag | The tag chain A should be upgraded to             |
| chain-binary      | The chain binary name                             |
| upgrade-plan-name | The name of the upgrade plan to execute           |
| test-entry-point  | Always TestUpgradeTestSuite                       |
| test              | Should be TestIBCChainUpgrade or TestChainUpgrade |

> TestIBCChainUpgrade should be used for ibc tests, while TestChainUpgrade should be used for single chain tests.
