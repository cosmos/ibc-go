//go:build !test_e2e

package transfer

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	feetypes "github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// compatibility:from_version: v9.0.0
func TestTransferChannelUpgradesV1TestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferChannelUpgradesV1TestSuite))
}

type TransferChannelUpgradesV1TestSuite struct {
	testsuite.E2ETestSuite
}

func (s *TransferChannelUpgradesV1TestSuite) SetupChannelUpgradesV1Test(testName string) {
	opts := s.TransferChannelOptions()
	opts.Version = transfertypes.V1
	s.CreatePaths(ibc.DefaultClientOpts(), opts, testName)
}

// TestChannelUpgrade_WithICS20v2_Succeeds tests upgrading a transfer channel to ICS20 v2.
func (s *TransferChannelUpgradesV1TestSuite) TestChannelUpgrade_WithICS20v2_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	s.SetupChannelUpgradesV1Test(testName)

	relayer, channelA := s.GetRelayerForTest(testName), s.GetChainAChannelForTest(testName)

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

	t.Run("verify transfer version of channel A is ics20-1", func(t *testing.T) {
		channel, err := query.Channel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(transfertypes.V1, channel.Version, "the channel version is not ics20-1")
	})

	t.Run("native token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		chainBWalletAmount := ibc.WalletAmount{
			Address: chainBWallet.FormattedAddress(), // destination address
			Denom:   chainA.Config().Denom,
			Amount:  sdkmath.NewInt(testvalues.IBCTransferAmount),
		}

		transferTxResp, err := chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName(), chainBWalletAmount, ibc.TransferOptions{})
		s.Require().NoError(err)
		s.Require().NoError(transferTxResp.Validate(), "chain-a ibc transfer tx is invalid")
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)

		actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainA, chainADenom)
		s.Require().NoError(err)

		expectedTotalEscrow := sdk.NewCoin(chainADenom, sdkmath.NewInt(testvalues.IBCTransferAmount))
		s.Require().Equal(expectedTotalEscrow, actualTotalEscrow)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	t.Run("execute gov proposal to initiate channel upgrade", func(t *testing.T) {
		chA, err := query.Channel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)

		upgradeFields := channeltypes.NewUpgradeFields(chA.Ordering, chA.ConnectionHops, transfertypes.V2)
		s.InitiateChannelUpgrade(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, upgradeFields)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB), "failed to wait for blocks")

	t.Run("verify channel A upgraded and transfer version is ics20-2", func(t *testing.T) {
		channel, err := query.Channel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(transfertypes.V2, channel.Version, "the channel version is not ics20-2")
	})

	t.Run("verify channel B upgraded and transfer version is ics20-2", func(t *testing.T) {
		channel, err := query.Channel(ctx, chainB, channelB.PortID, channelB.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(transfertypes.V2, channel.Version, "the channel version is not ics20-2")
	})

	// send the native chainB denom and also the ibc token from chainA
	transferCoins := []sdk.Coin{
		testvalues.DefaultTransferAmount(chainBIBCToken.IBCDenom()),
		testvalues.DefaultTransferAmount(chainBDenom),
	}

	t.Run("native token from chain B and non-native IBC token from chainA, both to chainA", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, transferCoins, chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "", nil)
		s.AssertTxSuccess(transferTxResp)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainB, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

		t.Run("chain A native denom", func(t *testing.T) {
			actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("chain B IBC denom", func(t *testing.T) {
			actualBalance, err := query.Balance(ctx, chainA, chainAAddress, chainAIBCToken.IBCDenom())
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance.Int64())
		})
	})

	t.Run("tokens are un-escrowed", func(t *testing.T) {
		t.Run("chain A escrow", func(t *testing.T) {
			actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainA, chainADenom)
			s.Require().NoError(err)
			s.Require().Equal(sdk.NewCoin(chainADenom, sdkmath.NewInt(0)), actualTotalEscrow) // total escrow is zero because tokens have come back
		})

		t.Run("chain B escrow", func(t *testing.T) {
			actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainB, chainBDenom)
			s.Require().NoError(err)
			s.Require().Equal(sdk.NewCoin(chainBDenom, sdkmath.NewInt(testvalues.IBCTransferAmount)), actualTotalEscrow)
		})
	})
}

// TestChannelUpgrade_WithFeeMiddlewareAndICS20v2_Succeeds tests upgrading a transfer channel to wire up fee middleware and upgrade to ICS20 v2.
func (s *TransferChannelUpgradesV1TestSuite) TestChannelUpgrade_WithFeeMiddlewareAndICS20v2_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	s.SetupChannelUpgradesV1Test(testName)

	relayer, channelA := s.GetRelayerForTest(testName), s.GetChainAChannelForTest(testName)

	channelB := channelA.Counterparty
	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom
	chainBDenom := chainB.Config().Denom
	chainAIBCToken := testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID)
	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelB.PortID, channelB.ChannelID)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	var (
		err     error
		channel channeltypes.Channel
	)

	t.Run("verify transfer version of channel A is ics20-1", func(t *testing.T) {
		channel, err = query.Channel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(transfertypes.V1, channel.Version, "the channel version is not ics20-1")
	})

	t.Run("native token transfer from chainB to chainA, sender is source of tokens", func(t *testing.T) {
		chainAwalletAmount := ibc.WalletAmount{
			Address: chainAWallet.FormattedAddress(), // destination address
			Denom:   chainBDenom,
			Amount:  sdkmath.NewInt(testvalues.IBCTransferAmount),
		}

		transferTxResp, err := chainB.SendIBCTransfer(ctx, channelB.ChannelID, chainBWallet.KeyName(), chainAwalletAmount, ibc.TransferOptions{})
		s.Require().NoError(err)
		s.Require().NoError(transferTxResp.Validate(), "chain-b ibc transfer tx is invalid")
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("execute gov proposal to initiate channel upgrade", func(t *testing.T) {
		channel.Version = transfertypes.V2 // change version to ics20-2
		s.InitiateChannelUpgrade(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, s.CreateUpgradeFields(channel))
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB), "failed to wait for blocks")

	t.Run("verify channel A upgraded and channel version is {ics29-1,ics20-2}", func(t *testing.T) {
		channel, err = query.Channel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)

		// check the channel version include the fee version
		version, err := feetypes.MetadataFromVersion(channel.Version)
		s.Require().NoError(err)
		s.Require().Equal(feetypes.Version, version.FeeVersion, "the channel version did not include ics29")
		s.Require().Equal(transfertypes.V2, version.AppVersion, "the channel version is not ics20-2")
	})

	t.Run("verify channel B upgraded and channel version is {ics29-1,ics20-2}", func(t *testing.T) {
		channel, err = query.Channel(ctx, chainB, channelB.PortID, channelB.ChannelID)
		s.Require().NoError(err)

		// check the channel version include the fee version
		version, err := feetypes.MetadataFromVersion(channel.Version)
		s.Require().NoError(err)
		s.Require().Equal(feetypes.Version, version.FeeVersion, "the channel version did not include ics29")
		s.Require().Equal(transfertypes.V2, version.AppVersion, "the channel version is not ics20-2")
	})

	var (
		chainARelayerWallet, chainBRelayerWallet ibc.Wallet
		relayerAStartingBalance                  int64
		testFee                                  = testvalues.DefaultFee(chainADenom)
	)

	t.Run("recover relayer wallets", func(t *testing.T) {
		_, _, err := s.RecoverRelayerWallets(ctx, relayer, testName)
		s.Require().NoError(err)

		chainARelayerWallet, chainBRelayerWallet, err = s.GetRelayerWallets(relayer)
		s.Require().NoError(err)

		relayerAStartingBalance, err = s.GetChainANativeBalance(ctx, chainARelayerWallet)
		s.Require().NoError(err)
		t.Logf("relayer A user starting with balance: %d", relayerAStartingBalance)
	})

	t.Run("register and verify counterparty payee", func(t *testing.T) {
		_, chainBRelayerUser := s.GetRelayerUsers(ctx, testName)
		resp := s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.FormattedAddress(), chainARelayerWallet.FormattedAddress())
		s.AssertTxSuccess(resp)

		address, err := query.CounterPartyPayee(ctx, chainB, chainBRelayerWallet.FormattedAddress(), channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.FormattedAddress(), address)
	})

	// send the native chainA denom and also the ibc token from chainB
	denoms := []string{chainAIBCToken.IBCDenom(), chainADenom}
	var transferCoins []sdk.Coin
	for _, denom := range denoms {
		transferCoins = append(transferCoins, testvalues.DefaultTransferAmount(denom))
	}

	t.Run("send incentivized transfer packet to chain B with native token from chain A and non-native IBC token from chainB", func(t *testing.T) {
		// before adding fees for the packet, there should not be incentivized packets
		packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)

		msgPayPacketFee := feetypes.NewMsgPayPacketFee(testFee, channelA.PortID, channelA.ChannelID, chainAWallet.FormattedAddress(), nil)
		msgTransfer := testsuite.GetMsgTransfer(
			channelA.PortID,
			channelA.ChannelID,
			transfertypes.V2,
			transferCoins,
			chainAWallet.FormattedAddress(),
			chainBWallet.FormattedAddress(),
			s.GetTimeoutHeight(ctx, chainB),
			0,
			"",
			nil,
		)
		resp := s.BroadcastMessages(ctx, chainA, chainAWallet, msgPayPacketFee, msgTransfer)
		s.AssertTxSuccess(resp)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		t.Run("chain B native denom", func(t *testing.T) {
			actualBalance, err := s.GetChainBNativeBalance(ctx, chainBWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("chain A IBC denom", func(t *testing.T) {
			actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance.Int64())
		})
	})

	t.Run("timeout fee is refunded", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount - testFee.AckFee.AmountOf(chainADenom).Int64() - testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("relayerA is paid ack and recv fee", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainARelayerWallet)
		s.Require().NoError(err)

		expected := relayerAStartingBalance + testFee.AckFee.AmountOf(chainADenom).Int64() + testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("tokens are un-escrowed", func(t *testing.T) {
		actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainB, chainADenom)
		s.Require().NoError(err)
		s.Require().Equal(sdk.NewCoin(chainADenom, sdkmath.NewInt(0)), actualTotalEscrow) // total escrow is zero because tokens have come back
	})
}
