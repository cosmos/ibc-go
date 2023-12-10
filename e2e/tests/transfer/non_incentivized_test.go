//go:build !test_e2e

package transfer

import (
	"context"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
)

func TestNonIncentivizedTransferTestSuite(t *testing.T) {
	testifysuite.Run(t, new(NonIncentivizedTransferTestSuite))
}

type NonIncentivizedTransferTestSuite struct {
	testsuite.E2ETestSuite
	chainA ibc.Chain
	chainB ibc.Chain
	rly    ibc.Relayer
}

func (s *NonIncentivizedTransferTestSuite) SetupTest() {
	ctx := context.TODO()
	s.chainA, s.chainB = s.GetChains()
	s.rly = s.SetupRelayer(ctx, s.TransferChannelOptions(), s.chainA, s.chainB)
}

// TestMsgTransfer_Succeeds_Nonincentivized will test sending successful IBC transfers from chainA to chainB.
// The transfer will occur over a basic transfer channel (non incentivized) and both native and non-native tokens
// will be sent forwards and backwards in the IBC transfer timeline (both chains will act as source and receiver chains).
func (s *NonIncentivizedTransferTestSuite) TestMsgTransfer_Succeeds_Nonincentivized() {
	t := s.T()

	ctx := context.TODO()

	channelA, err := s.rly.GetChannels(ctx, s.GetRelayerExecReporter(), s.chainA.Config().ChainID)
	s.Require().NoError(err)
	chainAChannels := channelA[len(channelA)-1]

	chainADenom := s.chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, s.chainA)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount, s.chainB)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, s.chainA, s.chainB), "failed to wait for blocks")
	t.Run("ensure capability module BeginBlock is executed", func(t *testing.T) {
		// by restarting the chain we ensure that the capability module's BeginBlocker is executed.
		s.Require().NoError(s.chainA.(*cosmos.CosmosChain).StopAllNodes(ctx))
		s.Require().NoError(s.chainA.(*cosmos.CosmosChain).StartAllNodes(ctx))
		s.Require().NoError(test.WaitForBlocks(ctx, 5, s.chainA), "failed to wait for blocks")
		s.InitGRPCClients(s.chainA)
	})

	chainAVersion := s.chainA.Config().Images[0].Version
	chainBVersion := s.chainB.Config().Images[0].Version

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, s.chainA, chainAWallet, chainAChannels.PortID, chainAChannels.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, s.chainB), 0, "", s.chainB)
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)

		if testvalues.TotalEscrowFeatureReleases.IsSupported(chainAVersion) {
			actualTotalEscrow, err := s.QueryTotalEscrowForDenom(ctx, s.chainA, chainADenom)
			s.Require().NoError(err)

			expectedTotalEscrow := sdk.NewCoin(chainADenom, sdkmath.NewInt(testvalues.IBCTransferAmount))
			s.Require().Equal(expectedTotalEscrow, actualTotalEscrow)
		}
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(s.rly)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, chainAChannels.Counterparty.PortID, chainAChannels.Counterparty.ChannelID)

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, s.chainA, chainAChannels.PortID, chainAChannels.ChannelID, 1)

		actualBalance, err := s.QueryBalance(ctx, s.chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	if testvalues.TokenMetadataFeatureReleases.IsSupported(chainBVersion) {
		t.Run("metadata for IBC denomination exists on chainB", func(t *testing.T) {
			s.AssertHumanReadableDenom(ctx, s.chainB, chainADenom, chainAChannels)
		})
	}

	t.Run("non-native IBC token transfer from chainB to chainA, receiver is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, s.chainB, chainBWallet, chainAChannels.Counterparty.PortID, chainAChannels.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBIBCToken.IBCDenom()), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, s.chainA), 0, "", s.chainA)
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.QueryBalance(ctx, s.chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		s.Require().Equal(sdkmath.ZeroInt(), actualBalance)

		if testvalues.TotalEscrowFeatureReleases.IsSupported(chainBVersion) {
			actualTotalEscrow, err := s.QueryTotalEscrowForDenom(ctx, s.chainB, chainBIBCToken.IBCDenom())
			s.Require().NoError(err)
			s.Require().Equal(sdk.NewCoin(chainBIBCToken.IBCDenom(), sdkmath.NewInt(0)), actualTotalEscrow) // total escrow is zero because sending chain is not source for tokens
		}
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, s.chainA, s.chainB), "failed to wait for blocks")

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, s.chainB, chainAChannels.Counterparty.PortID, chainAChannels.Counterparty.ChannelID, 1)

		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance)
	})

	if testvalues.TotalEscrowFeatureReleases.IsSupported(chainAVersion) {
		t.Run("tokens are un-escrowed", func(t *testing.T) {
			actualTotalEscrow, err := s.QueryTotalEscrowForDenom(ctx, s.chainA, chainADenom)
			s.Require().NoError(err)
			s.Require().Equal(sdk.NewCoin(chainADenom, sdkmath.NewInt(0)), actualTotalEscrow) // total escrow is zero because tokens have come back
		})
	}
}

func (s *NonIncentivizedTransferTestSuite) TestMsgTransfer_Timeout_Nonincentivized() {
	t := s.T()

	ctx := context.TODO()

	channelA, err := s.rly.GetChannels(ctx, s.GetRelayerExecReporter(), s.chainA.Config().ChainID)
	s.Require().NoError(err)
	chainAChannels := channelA[len(channelA)-1]

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, s.chainA)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount, s.chainB)

	chainBWalletAmount := ibc.WalletAmount{
		Address: chainBWallet.FormattedAddress(), // destination address
		Denom:   s.chainA.Config().Denom,
		Amount:  sdkmath.NewInt(testvalues.IBCTransferAmount),
	}

	t.Run("IBC transfer packet timesout", func(t *testing.T) {
		tx, err := s.chainA.SendIBCTransfer(ctx, chainAChannels.ChannelID, chainAWallet.KeyName(), chainBWalletAmount, ibc.TransferOptions{Timeout: testvalues.ImmediatelyTimeout()})
		s.Require().NoError(err)
		s.Require().NoError(tx.Validate(), "source ibc transfer tx is invalid")
		time.Sleep(time.Nanosecond * 1) // want it to timeout immediately
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(s.rly)
	})

	t.Run("ensure escrowed tokens have been refunded to sender due to timeout", func(t *testing.T) {
		// ensure destination address did not receive any tokens
		bal, err := s.GetChainBNativeBalance(ctx, chainBWallet, s.chainB)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, bal)

		// ensure that the sender address has been successfully refunded the full amount
		bal, err = s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, bal)
	})
}
