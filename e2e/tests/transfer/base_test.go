//go:build !test_e2e

package transfer

import (
	"context"
	"testing"
	"time"

	"github.com/cosmos/interchaintest/v10/ibc"
	test "github.com/cosmos/interchaintest/v10/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

// compatibility:from_version: v7.10.0
func TestTransferTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferTestSuite))
}

type TransferTestSuite struct {
	transferTester
}

// SetupSuite sets up chains for the current test suite
func (s *TransferTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil)
}

// transferTester defines some helper functions that can be used in various test suites
// that test transfer functionality.
type transferTester struct {
	testsuite.E2ETestSuite
}

// QueryTransferParams queries the on-chain send enabled param for the transfer module
func (s *transferTester) QueryTransferParams(ctx context.Context, chain ibc.Chain) transfertypes.Params {
	res, err := query.GRPCQuery[transfertypes.QueryParamsResponse](ctx, chain, &transfertypes.QueryParamsRequest{})
	s.Require().NoError(err)
	return *res.Params
}

// TestMsgTransfer_Succeeds_Nonincentivized will test sending successful IBC transfers from chainA to chainB.
// The transfer will occur over a basic transfer channel (non incentivized) and both native and non-native tokens
// will be sent forwards and backwards in the IBC transfer timeline (both chains will act as source and receiver chains).
func (s *TransferTestSuite) TestMsgTransfer_Succeeds_Nonincentivized() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()

	// NOTE: t.Parallel() should be called before SetupPath in all tests.
	// t.Name() must be stored in a variable before t.Parallel() otherwise t.Name() is not
	// deterministic.
	t.Parallel()

	chainA, chainB := s.GetChains()

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	chainBVersion := chainB.Config().Images[0].Version
	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)

		// TODO: cannot query total escrow if tests in parallel are using the same denom.
		// if testvalues.TotalEscrowFeatureReleases.IsSupported(chainAVersion) {
		//	actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainA, chainADenom)
		//	s.Require().NoError(err)
		//
		//	expectedTotalEscrow := sdk.NewCoin(chainADenom, sdkmath.NewInt(testvalues.IBCTransferAmount))
		//	s.Require().Equal(expectedTotalEscrow, actualTotalEscrow)
		// }
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	if testvalues.TokenMetadataFeatureReleases.IsSupported(chainBVersion) {
		t.Run("metadata for IBC denomination exists on chainB", func(t *testing.T) {
			s.AssertHumanReadableDenom(ctx, chainB, chainADenom, channelA)
		})
	}

	t.Run("non-native IBC token transfer from chainB to chainA, receiver is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBIBCToken.IBCDenom()), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		s.Require().Equal(sdkmath.ZeroInt(), actualBalance)

		// https://github.com/cosmos/ibc-go/issues/6742
		// if testvalues.TotalEscrowFeatureReleases.IsSupported(chainBVersion) {
		//	actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainB, chainBIBCToken.IBCDenom())
		//	s.Require().NoError(err)
		//	s.Require().Equal(sdk.NewCoin(chainBIBCToken.IBCDenom(), sdkmath.NewInt(0)), actualTotalEscrow) // total escrow is zero because sending chain is not source for tokens
		// }
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainB, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance)
	})

	// https://github.com/cosmos/ibc-go/issues/6742
	// if testvalues.TotalEscrowFeatureReleases.IsSupported(chainAVersion) {
	//	t.Run("tokens are un-escrowed", func(t *testing.T) {
	//		actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainA, chainADenom)
	//		s.Require().NoError(err)
	//		s.Require().Equal(sdk.NewCoin(chainADenom, sdkmath.NewInt(0)), actualTotalEscrow) // total escrow is zero because tokens have come back
	//	})
	// }
}

// TestMsgTransfer_Fails_InvalidAddress attempts to send an IBC transfer to an invalid address and ensures
// that the tokens on the sending chain are unescrowed.
func (s *TransferTestSuite) TestMsgTransfer_Fails_InvalidAddress() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	t.Parallel()

	chainA, chainB := s.GetChains()

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("native IBC token transfer from chainA to invalid address", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, testvalues.InvalidAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)
	})

	t.Run("token transfer amount unescrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance)
	})
}

func (s *TransferTestSuite) TestMsgTransfer_Timeout_Nonincentivized() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	t.Parallel()

	chainA, chainB := s.GetChains()

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	chainBWalletAmount := ibc.WalletAmount{
		Address: chainBWallet.FormattedAddress(), // destination address
		Denom:   chainA.Config().Denom,
		Amount:  sdkmath.NewInt(testvalues.IBCTransferAmount),
	}

	t.Run("IBC transfer packet timesout", func(t *testing.T) {
		tx, err := chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName(), chainBWalletAmount, ibc.TransferOptions{Timeout: testvalues.ImmediatelyTimeout()})
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
		s.StartRelayer(relayer, testName)
	})

	t.Run("ensure escrowed tokens have been refunded to sender due to timeout", func(t *testing.T) {
		// ensure destination address did not receive any tokens
		bal, err := s.GetChainBNativeBalance(ctx, chainBWallet)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, bal)

		// ensure that the sender address has been successfully refunded the full amount
		bal, err = s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, bal)
	})
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

	testName := t.Name()
	t.Parallel()

	chainA, chainB := s.GetChains()

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("IBC token transfer with memo from chainA to chainB", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "memo")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("packets relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)
		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())

		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, actualBalance.Int64())
	})
}

// TestMsgTransfer_EntireBalance tests that it is possible to transfer the entire balance
// of a given denom by using types.UnboundedSpendLimit as the amount.
// compatibility:TestMsgTransfer_EntireBalance:from_versions: v7.10.0,v8.7.0,v10.0.0
func (s *TransferTestSuite) TestMsgTransfer_EntireBalance() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	t.Parallel()

	chainA, chainB := s.GetChains()

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	coinFromA := testvalues.DefaultTransferAmount(chainADenom)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("IBC token transfer from chainA to chainB", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, coinFromA, chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		s.Require().Equal(testvalues.StartingTokenAmount-coinFromA.Amount.Int64(), actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)
	t.Run("packets relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)
		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())

		s.Require().NoError(err)
		s.Require().Equal(coinFromA.Amount.Int64(), actualBalance.Int64())

		actualBalance, err = query.Balance(ctx, chainA, chainAAddress, chainADenom)

		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount-coinFromA.Amount.Int64(), actualBalance.Int64())
	})

	t.Run("send entire balance from B to A", func(t *testing.T) {
		transferCoin := sdk.NewCoin(chainBIBCToken.IBCDenom(), transfertypes.UnboundedSpendLimit())

		transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, transferCoin, chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("packets relayed", func(t *testing.T) {
		// test that chainA has the entire balance back of its native token.
		s.AssertPacketRelayed(ctx, chainB, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)
		actualBalance, err := query.Balance(ctx, chainA, chainAAddress, chainADenom)

		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, actualBalance.Int64())

		// test that chainB has a zero balance of chainA's IBC token denom.
		actualBalance, err = query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())

		s.Require().NoError(err)
		s.Require().Zero(actualBalance.Int64())
	})
}
