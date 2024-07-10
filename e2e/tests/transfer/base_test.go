//go:build !test_e2e

package transfer

import (
	"context"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
)

func TestTransferTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferTestSuite))
}

type TransferTestSuite struct {
	transferTester
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

// CreateTransferPath sets up a path between chainA and chainB with a transfer channel and returns the relayer wired
// up to watch the channel and port IDs created.
func (s *transferTester) CreateTransferPath(testName string) (ibc.Relayer, ibc.ChannelOutput) {
	relayer, channel := s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName), s.GetChainAChannelForTest(testName)
	s.T().Logf("test %s running on portID %s channelID %s", testName, channel.PortID, channel.ChannelID)
	return relayer, channel
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

	relayer, channelA := s.CreateTransferPath(testName)

	chainA, chainB := s.GetChains()

	chainBVersion := chainB.Config().Images[0].Version
	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")
	// TODO: https://github.com/cosmos/ibc-go/issues/6743
	// t.Run("ensure capability module BeginBlock is executed", func(t *testing.T) {
	//	// by restarting the chain we ensure that the capability module's BeginBlocker is executed.
	//	s.Require().NoError(chainA.(*cosmos.CosmosChain).StopAllNodes(ctx))
	//	s.Require().NoError(chainA.(*cosmos.CosmosChain).StartAllNodes(ctx))
	//	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA), "failed to wait for blocks")
	// })

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
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
		transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferCoins(chainBIBCToken.IBCDenom()), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
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

// TestMsgTransfer_Succeeds_MultiDenom will test sending successful IBC transfers from chainA to chainB.
// A multidenom transfer with native chainB tokens and IBC tokens from chainA is executed from chainB to chainA.
func (s *TransferTestSuite) TestMsgTransfer_Succeeds_Nonincentivized_MultiDenom() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	t.Parallel()
	relayer, channelA := s.CreateTransferPath(testName)

	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom
	chainBDenom := chainB.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)
	chainAIBCToken := testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID)

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, sdk.NewCoins(testvalues.DefaultTransferAmount(chainADenom)), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("native chainA tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)

		// https://github.com/cosmos/ibc-go/issues/6742
		// actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainA, chainADenom)
		// s.Require().NoError(err)
		//
		// expectedTotalEscrow := sdk.NewCoin(chainADenom, sdkmath.NewInt(testvalues.IBCTransferAmount))
		// s.Require().Equal(expectedTotalEscrow, actualTotalEscrow)
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

	t.Run("metadata for IBC denomination exists on chainB", func(t *testing.T) {
		s.AssertHumanReadableDenom(ctx, chainB, chainADenom, channelA)
	})

	// send the native chainB denom and also the ibc token from chainA
	transferCoins := []sdk.Coin{
		testvalues.DefaultTransferAmount(chainBIBCToken.IBCDenom()),
		testvalues.DefaultTransferAmount(chainBDenom),
	}

	t.Run("native token from chain B and non-native IBC token from chainA, both to chainA", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, transferCoins, chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
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

	// https://github.com/cosmos/ibc-go/issues/6742
	// t.Run("native chainA tokens are un-escrowed", func(t *testing.T) {
	//	actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainA, chainADenom)
	//	s.Require().NoError(err)
	//	s.Require().Equal(sdk.NewCoin(chainADenom, sdkmath.NewInt(0)), actualTotalEscrow) // total escrow is zero because tokens have come back
	// })
}

// TestMsgTransfer_Fails_InvalidAddress_MultiDenom attempts to send a multidenom IBC transfer
// to an invalid address and ensures that the tokens on the sending chain are returned to the sender.
func (s *TransferTestSuite) TestMsgTransfer_Fails_InvalidAddress_MultiDenom() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	t.Parallel()
	relayer, channelA := s.CreateTransferPath(testName)

	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom
	chainBDenom := chainB.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, sdk.NewCoins(testvalues.DefaultTransferAmount(chainADenom)), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("native chainA tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)

		// https://github.com/cosmos/ibc-go/issues/6742
		// actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainA, chainADenom)
		// s.Require().NoError(err)
		//
		// expectedTotalEscrow := sdk.NewCoin(chainADenom, sdkmath.NewInt(testvalues.IBCTransferAmount))
		// s.Require().Equal(expectedTotalEscrow, actualTotalEscrow)
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

	t.Run("metadata for IBC denomination exists on chainB", func(t *testing.T) {
		s.AssertHumanReadableDenom(ctx, chainB, chainADenom, channelA)
	})

	// send the native chainB denom and also the ibc token from chainA
	transferCoins := []sdk.Coin{
		testvalues.DefaultTransferAmount(chainBIBCToken.IBCDenom()),
		testvalues.DefaultTransferAmount(chainBDenom),
	}

	t.Run("stop relayer", func(t *testing.T) {
		s.StopRelayer(ctx, relayer)
	})

	t.Run("native token from chain B and non-native IBC token from chainA, both to chainA", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, transferCoins, chainBAddress, testvalues.InvalidAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("tokens are sent from chain B", func(t *testing.T) {
		t.Run("native chainB tokens are escrowed", func(t *testing.T) {
			actualBalance, err := s.GetChainBNativeBalance(ctx, chainBWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("non-native chainA IBC denom are burned", func(t *testing.T) {
			actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
			s.Require().NoError(err)
			s.Require().Equal(int64(0), actualBalance.Int64())
		})
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainB, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)
	})

	t.Run("tokens are returned to sender on chainB", func(t *testing.T) {
		t.Run("native chainB denom", func(t *testing.T) {
			actualBalance, err := s.GetChainBNativeBalance(ctx, chainBWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("non-native chainA IBC denom", func(t *testing.T) {
			actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance.Int64())
		})
	})
}

// TestMsgTransfer_Fails_InvalidAddress attempts to send an IBC transfer to an invalid address and ensures
// that the tokens on the sending chain are unescrowed.
func (s *TransferTestSuite) TestMsgTransfer_Fails_InvalidAddress() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	t.Parallel()
	relayer, channelA := s.CreateTransferPath(testName)

	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("native IBC token transfer from chainA to invalid address", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, testvalues.InvalidAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
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
	relayer, channelA := s.CreateTransferPath(testName)

	chainA, _ := s.GetChains()

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
	relayer, channelA := s.CreateTransferPath(testName)

	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("IBC token transfer with memo from chainA to chainB", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "memo")
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
