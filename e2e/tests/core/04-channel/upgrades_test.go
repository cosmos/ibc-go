//go:build !test_e2e

package channel

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	"github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	feetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

func TestChannelTestSuite(t *testing.T) {
	testifysuite.Run(t, new(ChannelTestSuite))
}

type ChannelTestSuite struct {
	testsuite.E2ETestSuite
}

// TestChannelUpgradeWithFeeMiddleware
func (s *ChannelTestSuite) TestChannelUpgradeWithFeeMiddleware() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.TransferChannelOptions())
	channelB := channelA.Counterparty
	chainA, chainB := s.GetChains()
	// chainAVersion := chainA.Config().Images[0].Version

	chainADenom := chainA.Config().Denom
	chainBDenom := chainB.Config().Denom
	chainAIBCToken := testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID)
	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelB.PortID, channelB.ChannelID)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	// trying to create some inflight packets, although they might get relayed before the upgrade starts
	t.Run("create inflight transfer packets between chain A and chain B", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)

		transferTxResp = s.Transfer(ctx, chainB, chainBWallet, channelB.PortID, channelB.ChannelID, testvalues.DefaultTransferAmount(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("execute gov proposal to initiate channel upgrade", func(t *testing.T) {
		s.initiateChannelUpgrade(ctx, chainA, chainAWallet, channelA)
	})

	// TODO: eventually we should be able to start the relayer after the gov proposal executes, but we need a new relayer image with a fix for this
	// t.Run("start relayer", func(t *testing.T) {
	// 	s.StartRelayer(relayer)
	// })

	s.Require().NoError(test.WaitForBlocks(ctx, 40, chainA, chainB), "failed to wait for blocks")

	t.Run("packets are relayed between chain A and chain B", func(t *testing.T) {
		// packet from chain A to chain B
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)
		actualBalance, err := s.QueryBalance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)
		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())

		// packet from chain B to chain A
		s.AssertPacketRelayed(ctx, chainB, channelB.PortID, channelB.ChannelID, 1)
		actualBalance, err = s.QueryBalance(ctx, chainA, chainAAddress, chainAIBCToken.IBCDenom())
		s.Require().NoError(err)
		expected = testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	t.Run("verify channel A upgraded and is fee enabled", func(t *testing.T) {
		channel, err := s.QueryChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)

		// check the channel version include the fee version
		version, err := feetypes.MetadataFromVersion(channel.Version)
		s.Require().NoError(err)
		s.Require().Equal(feetypes.Version, version, "the channel version did not include ics29")

		// extra check
		feeEnabled, err := s.QueryFeeEnabledChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(true, feeEnabled)
	})

	t.Run("verify channel B upgraded and has version with ics29", func(t *testing.T) {
		channel, err := s.QueryChannel(ctx, chainB, channelB.PortID, channelB.ChannelID)
		s.Require().NoError(err)

		// check the channel version include the fee version
		version, err := feetypes.MetadataFromVersion(channel.Version)
		s.Require().NoError(err)
		s.Require().Equal(feetypes.Version, version, "the channel version did not include ics29")

		// extra check
		feeEnabled, err := s.QueryFeeEnabledChannel(ctx, chainB, channelB.PortID, channelB.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(true, feeEnabled)
	})

}

// initiateChannelUpgrade
func (s *ChannelTestSuite) initiateChannelUpgrade(ctx context.Context, chain ibc.Chain, wallet ibc.Wallet, channel ibc.ChannelOutput) {
	govModuleAddress, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chain)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	versionMetadata := feetypes.Metadata{
		FeeVersion: feetypes.Version,
		AppVersion: transfertypes.Version,
	}
	versionBytes, err := types.ModuleCdc.MarshalJSON(&versionMetadata)
	s.Require().NoError(err)

	upgradeFields := channeltypes.NewUpgradeFields(channeltypes.UNORDERED, channel.ConnectionHops, string(versionBytes))
	msg := channeltypes.NewMsgChannelUpgradeInit(channel.PortID, channel.ChannelID, upgradeFields, govModuleAddress.String())
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, wallet)
}
