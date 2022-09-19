package transfer

import (
	"context"
	"fmt"
	"testing"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	"github.com/strangelove-ventures/ibctest/test"
)

// This can be used to test sending with a packet enhanced with Metadata for different combinations of
// versions.
// To do this, first we need to build the versions and tag them:
// > git checkout <the branch with the metadata implementation>
// > docker build . -t local:latest
// > git checkout v3.2.1
// > docker build . -t local:v3.2.1
//
// Then we can run the tests:
//
//     CHAIN_IMAGE=local CHAIN_A_TAG="v3.2.1" CHAIN_B_TAG="v3.2.1" make e2e-test entrypoint=TransferTestSuite test=TestMsgTransfer_WithMetadata
//     CHAIN_IMAGE=local CHAIN_A_TAG="v3.2.1" CHAIN_B_TAG="latest" make e2e-test entrypoint=TransferTestSuite test=TestMsgTransfer_WithMetadata
//     CHAIN_IMAGE=local CHAIN_A_TAG="latest" CHAIN_B_TAG="v3.2.1" make e2e-test entrypoint=TransferTestSuite test=TestMsgTransfer_WithMetadata
//     CHAIN_IMAGE=local CHAIN_A_TAG="latest" CHAIN_B_TAG="latest" make e2e-test entrypoint=TransferTestSuite test=TestMsgTransfer_WithMetadata
//
// All of the above combinations should pass.

// TestMsgTransfer_WithMetadata will test sending IBC transfers from chainA to chainB
// If the chains contain a version of FungibleTokenPacketData with metadata, both send and receive should succeed.
// If one of the chains contains a version of FungibleTokenPacketData without metadata, then receiving a packet with
// metadata should fail in that chain
func (s *TransferTestSuite) TestMsgTransfer_WithMetadata() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.Bech32Address(chainB.Config().Bech32Prefix)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	chainAVersion := chainA.Config().Images[0].Version
	chainBVersion := chainB.Config().Images[0].Version
	if chainAVersion != "latest" && chainAVersion != "v3.2.1" {
		s.Fail("Invalid chain version for chain A: %s", chainAVersion)
	}
	if chainBVersion != "latest" && chainBVersion != "v3.2.1" {
		s.Fail("Invalid chain version for chain B: %s", chainAVersion)
	}

	t.Logf("Running metadata tests versions chainA: %s, chainB: %s", chainAVersion, chainBVersion)

	t.Run("IBC token transfer with metadata from chainA to chainB", func(t *testing.T) {
		// this should only pass if the sender chain understands how to create a transfer with metadata
		transferTxResp, err := s.TransferWithMetadata(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, []byte("metadata"))
		fmt.Println(transferTxResp, err)
		if chainAVersion == "latest" {
			t.Logf("ChainA understands metadata. The send should succeed")
			s.Require().NoError(err)
			s.AssertValidTxResponse(transferTxResp)
		} else if chainAVersion == "v3.2.1" {
			t.Logf("ChainA does not understands metadata. The send should fail")
			s.Require().NoError(err)
			s.Require().Equal(transferTxResp.Code, uint32(2))
			s.Require().Contains(transferTxResp.RawLog, "errUnknownField")
		}
	})

	if chainAVersion == "v3.2.1" {
		// The send failed above
		return
	}

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("packets relayed?", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		if chainBVersion == "latest" {
			t.Logf("ChainB understands metadata. Receive should succeed")
			expected := testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance)
		} else if chainBVersion == "v3.2.1" {
			t.Logf("ChainB does not understands metadata. Receive should fail")
			s.Require().Equal(int64(0), actualBalance)
			return
		}

	})
}

// TestMsgTransfer_WithoutMetadata should always succeed.
func (s *TransferTestSuite) TestMsgTransfer_WithoutMetadata() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.Bech32Address(chainB.Config().Bech32Prefix)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	chainAVersion := chainA.Config().Images[0].Version
	chainBVersion := chainB.Config().Images[0].Version
	if chainAVersion != "latest" && chainAVersion != "v3.2.1" {
		s.Fail("Invalid chain version for chain A: %s", chainAVersion)
	}
	if chainBVersion != "latest" && chainBVersion != "v3.2.1" {
		s.Fail("Invalid chain version for chain A: %s", chainAVersion)
	}

	t.Logf("Running metadata tests versions chainA: %s, chainB: %s", chainAVersion, chainBVersion)

	t.Run("IBC token transfer with metadata from chainA to chainB", func(t *testing.T) {
		// this should only pass if the sender chain understands how to create a transfer with metadata
		transferTxResp, err := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0)
		s.Require().NoError(err)
		s.AssertValidTxResponse(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("packets relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)

	})
}
