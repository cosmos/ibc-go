package transfer

import (
	"context"
	"testing"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	"github.com/strangelove-ventures/ibctest/test"
)

// featureReleases represents the releases the memo field was released in.
var featureRelease = testsuite.FeatureReleases{
	MajorVersion:  "v6",
	MinorVersions: []string{"v2.5, v3.4, v4.2, v5.1"},
}

// This can be used to test sending with a transfer packet with a memo given different combinations of
// ibc-go versions.
//
// TestMsgTransfer_WithMemo will test sending IBC transfers from chainA to chainB
// If the chains contain a version of FungibleTokenPacketData with memo, both send and receive should succeed.
// If one of the chains contains a version of FungibleTokenPacketData without memo, then receiving a packet with
// memo should fail in that chain
func (s *TransferTestSuite) TestMsgTransfer_WithMemo() {
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

	t.Logf("Running memo tests versions chainA: %s, chainB: %s", chainAVersion, chainBVersion)

	t.Run("IBC token transfer with memo from chainA to chainB", func(t *testing.T) {
		transferTxResp, err := s.TransferWithMemo(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, []byte("memo"))
		s.Require().NoError(err)

		if !featureRelease.IsSupported(chainAVersion) {
			s.Require().Equal(transferTxResp.Code, uint32(2))
			s.Require().Contains(transferTxResp.RawLog, "errUnknownField")

			// transfer not sent, end test
			return
		}

		// sender chain supports feature
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

	t.Run("packets relayed?", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		if !featureRelease.IsSupported(chainBVersion) {
			s.Require().Equal(int64(0), actualBalance)

			// receive failed, end test
			return
		}

		// receving chain supports feature, transfer successful
		s.Require().Equal(testvalues.IBCTransferAmount, actualBalance)
	})
}
