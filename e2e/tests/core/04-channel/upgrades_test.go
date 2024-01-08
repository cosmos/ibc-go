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

// TestChannelUpgrade_WithFeeMiddleware_Succeeds tests upgrading a transfer channel to wire up fee middleware
func (s *ChannelTestSuite) TestChannelUpgrade_WithFeeMiddleware_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.TransferChannelOptions())
	channelB := channelA.Counterparty
	chainA, chainB := s.GetChains()

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

	t.Run("execute gov proposal to initiate channel upgrade", func(t *testing.T) {
		chA, err := s.QueryChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)

		s.initiateChannelUpgrade(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, s.createUpgradeFields(chA))
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB), "failed to wait for blocks")

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
		s.Require().Equal(feetypes.Version, version.FeeVersion, "the channel version did not include ics29")

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
		s.Require().Equal(feetypes.Version, version.FeeVersion, "the channel version did not include ics29")

		// extra check
		feeEnabled, err := s.QueryFeeEnabledChannel(ctx, chainB, channelB.PortID, channelB.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(true, feeEnabled)
	})
}

// TestChannelUpgrade_WithFeeMiddleware_FailsWithTimeoutOnAck tests upgrading a transfer channel to wire up fee middleware but fails on ACK because of timeout
func (s *ChannelTestSuite) TestChannelUpgrade_WithFeeMiddleware_FailsWithTimeoutOnAck() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.TransferChannelOptions())
	channelB := channelA.Counterparty
	chainA, chainB := s.GetChains()

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("execute gov proposal to set upgrade timeout", func(t *testing.T) {
		s.setUpgradeTimeoutParam(ctx, chainB, chainBWallet)
	})

	t.Run("execute gov proposal to initiate channel upgrade", func(t *testing.T) {
		chA, err := s.QueryChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)

		s.initiateChannelUpgrade(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, s.createUpgradeFields(chA))
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB), "failed to wait for blocks")

	t.Run("verify channel A did not upgrade", func(t *testing.T) {
		channel, err := s.QueryChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)

		s.Require().Equal(channeltypes.OPEN, channel.State, "the channel state is not OPEN")
		s.Require().Equal(transfertypes.Version, channel.Version, "the channel version is not ics20-v1")

		errorReceipt, err := s.QueryUpgradeError(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(uint64(1), errorReceipt.Sequence)
		s.Require().Contains(errorReceipt.Message, "restored channel to pre-upgrade state")
	})

	t.Run("verify channel B did not upgrade", func(t *testing.T) {
		channel, err := s.QueryChannel(ctx, chainB, channelB.PortID, channelB.ChannelID)
		s.Require().NoError(err)

		s.Require().Equal(channeltypes.OPEN, channel.State, "the channel state is not OPEN")
		s.Require().Equal(transfertypes.Version, channel.Version, "the channel version is not ics20-v1")

		errorReceipt, err := s.QueryUpgradeError(ctx, chainB, channelB.PortID, channelB.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(uint64(1), errorReceipt.Sequence)
		s.Require().Contains(errorReceipt.Message, "restored channel to pre-upgrade state")
	})
}

// createUpgradeFields created the upgrade fields for channel
func (s *ChannelTestSuite) createUpgradeFields(channel channeltypes.Channel) channeltypes.UpgradeFields {
	versionMetadata := feetypes.Metadata{
		FeeVersion: feetypes.Version,
		AppVersion: transfertypes.Version,
	}
	versionBytes, err := feetypes.ModuleCdc.MarshalJSON(&versionMetadata)
	s.Require().NoError(err)

	return channeltypes.NewUpgradeFields(channel.Ordering, channel.ConnectionHops, string(versionBytes))
}

// setUpgradeTimeoutParam creates and submits a governance proposal to execute the message to update 04-channel params with a timeout of 1s
func (s *ChannelTestSuite) setUpgradeTimeoutParam(ctx context.Context, chain ibc.Chain, wallet ibc.Wallet) {
	govModuleAddress, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chain)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	upgradeTimeout := channeltypes.NewTimeout(channeltypes.DefaultTimeout.Height, 1000000000)
	msg := channeltypes.NewMsgUpdateChannelParams(govModuleAddress.String(), channeltypes.NewParams(upgradeTimeout))
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, wallet)
}

// initiateChannelUpgrade creates and submits a governance proposal to execute the message to initiate a channel upgrade
func (s *ChannelTestSuite) initiateChannelUpgrade(ctx context.Context, chain ibc.Chain, wallet ibc.Wallet, portID, channelID string, upgradeFields channeltypes.UpgradeFields) {
	govModuleAddress, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chain)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	msg := channeltypes.NewMsgChannelUpgradeInit(portID, channelID, upgradeFields, govModuleAddress.String())
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, wallet)
}
