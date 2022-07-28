
## Table of Contents
1. [How to write tests](#how-to-write-tests)
   - a. [Adding a new test](#adding-a-new-test)
   - b. [Running tests locally](#running-tests-locally)
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
   - a. [ibctest](#ibctest)
   - b. [CI configuration](#ci-configuration)
3. [Troubleshooting](#troubleshooting)


## How to write tests

### Adding a new test

All tests should go under the [e2e](https://github.com/cosmos/ibc-go/tree/main/e2e) directory. When adding a new test, either add a new test function
to an existing test suite **_in the same file_**, or create a new test suite in a new file and add test functions there.
New test files should follow the convention of `module_name_test.go`.

New test suites should be composed of `testsuite.E2ETestSuite`. This type has lots of useful helper functionality that will
be quite common in most tests.

> Note: see [here](#how-tests-are-run) for details about these requirements.


### Running tests locally

Tests can be run using a Makefile target under the e2e directory. `e2e/Makefile`

There are several envinronment variables that alter the behaviour of the make target.

| Environment Variable      | Description | Default Value|
| ----------- | ----------- | ----------- |
| SIMD_IMAGE      | The image that will be used for simd       | ibc-go-simd-e2e |
| SIMD_TAG   | The tag used for simd        | latest|
| RLY_TAG   | The tag used for the go relayer        | main|


> Note: when running tests locally, **no images are pushed** to the `ghcr.io/cosmos/ibc-go-simd-e2e` registry.
The images which are used only exist on your machine.

These environment variables allow us to run tests with arbitrary verions (from branches or released) of simd
and the go relayer.

Every time changes are pushed to a branch or to `main`, a new `simd` image is built and pushed [here](https://github.com/cosmos/ibc-go/pkgs/container/ibc-go-simd-e2e).


#### Example Command:

```sh
export SIMD_IMAGE="ghcr.io/cosmos/ibc-go-simd-e2e"
export SIMD_TAG="pr-1650"
export RLY_TAG="v2.0.0-rc2"
make e2e-test suite=FeeMiddlewareTestSuite test=TestMultiMsg_MsgPayPacketFeeSingleSender
```


> Note: sometimes it can be useful to make changes to [ibctest](https://github.com/strangelove-ventures/ibctest) when running tests locally. In order to do this, add the following line to
e2e/go.mod

`replace github.com/strangelove-ventures/ibctest => ../ibctest`

Or point it to any local checkout you have.

### Code samples

#### Setup

Every standard test will start with this. This creates two chains and a relayer,
initializes relayer accounts on both chains, establishes a connection and a channel
between the chains.

Both chains have started, but the relayer is not yet started.

The relayer should be started as part of the test if required. See [Starting the Relayer](#starting-the-relayer)

```go
relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
chainA, chainB := s.GetChains()
```

#### Creating test users

There are helper functions to easily create users on both chains.

```go
chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
```

#### Waiting

We can wait for some number of blocks on the specified chains if required.

```go
chainA, chainB := s.GetChains()
err := test.WaitForBlocks(ctx, 1, chainA, chainB)
s.Require().NoError(err)
```

#### Query wallet balances

We can fetch balances of wallets on specific chains.

```go
chainABalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
s.Require().NoError(err)
```

#### Broadcasting messages

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

#### Starting the relayer

The relayer can be started with the following.

```go
t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
})
```

#### Arbitrary commands

Arbitrary commands can be executed on a given chain.

> Note: these commands will be fully configured to run on the chain executed on (home directory, ports configured etc.)

However, it is preferable to [broadcast messages](#broadcasting-messages) or use a gRPC query if possible.

```go
stdout, stderr, err := chainA.Exec(ctx, []string{"tx", "..."}, nil)
```

#### IBC transfer

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


#### ibctest

These E2E tests use the [ibctest framework](https://github.com/strangelove-ventures/ibctest). This framework creates chains and relayers in containers and allows for arbitrary commands to be executed in the chain containers,
as well as allowing us to broadcast arbitrary messages which are signed on behalf of a user created in the test.


#### CI configuration

There are two main github actions for e2e tests.

[e2e.yaml](https://github.com/cosmos/ibc-go/blob/main/.github/workflows/e2e.yaml) which runs when collaborators create branches.

[e2e-fork.yaml](https://github.com/cosmos/ibc-go/blob/main/.github/workflows/e2e-fork.yml) which runs when forks are created.

In `e2e.yaml`, the `simd` image is built and pushed to [a registry](https://github.com/cosmos/ibc-go/pkgs/container/ibc-go-simd-e2e) and every test
that is run uses the image that was built.

In `e2e-fork.yaml`, images are not pushed to this registry, but instead remain local to the host runner.


### How tests are run

The tests use the `matrix` feature of Github Actions. The matrix is
dynamically generated using [this command](https://github.com/cosmos/ibc-go/blob/main/cmd/build_test_matrix/main.go).

> Note: there is currently a limitation that all tests belonging to a test suite must be in the same file.
In order to support test functions spread in different files, we would either need to manually maintain a matrix
or update the script to account for this. The script assumes there is a single test suite per test file to avoid an overly complex
generation process.

Which looks under the `e2e` directory, and creates a task for each test suite function.

#### Example

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
         "suite": "FeeMiddlewareTestSuite",
         "test": "TestA"
      },
      {
         "suite": "FeeMiddlewareTestSuite",
         "test": "TestB"
      },
      {
         "suite": "FeeMiddlewareTestSuite",
         "test": "TestC"
      },
      {
         "suite": "TransferTestSuite",
         "test": "TestD"
      },
      {
         "suite": "TransferTestSuite",
         "test": "TestE"
      },
      {
         "suite": "TransferTestSuite",
         "test": "TestF"
      }
   ]
}
```

This string is used to generate a test matrix in the Github Action that runs the E2E tests.

All tests will be run on different hosts.


#### Misceleneous:

* Gas fees are set to zero to simply calcuations when asserting account balances.
* When upgrading from e.g. v4 -> v5, in ibc-go, we cannot upgrade the go.mod under `e2e` since v5 will not yet exist. We need to upgrade it in a follow up PR.

### Troubleshooting

* On Mac, after running a lot of tests, it can happen that containers start failing. To fix this, you can try clearing existing containers and restarting the docker daemon.

  This generally manifests itself as relayer or simd containers timing out during setup stages of the test. This doesn't happen in CI.
  ```bash
  # delete all images
  docker system prune -af
  ```
  This issue doesn't seem to occur on other operating systems.

  
