package transfer

import (
	"context"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"

	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	test "github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func TestTransferTestSuite(t *testing.T) {
	suite.Run(t, new(TransferTestSuite))
}

type TransferTestSuite struct {
	testsuite.E2ETestSuite
}

// QueryTransferSendEnabledParam queries the on-chain send enabled param for the transfer module
func (s *TransferTestSuite) QueryTransferParams(ctx context.Context, chain ibc.Chain) transfertypes.Params {
	queryClient := s.GetChainGRCPClients(chain).TransferQueryClient
	res, err := queryClient.Params(ctx, &transfertypes.QueryParamsRequest{})
	s.Require().NoError(err)
	return *res.Params
}

// TestMsgTransfer_Succeeds_Nonincentivized will test sending successful IBC transfers from chainA to chainB.
// The transfer will occur over a basic transfer channel (non incentivized) and both native and non-native tokens
// will be sent forwards and backwards in the IBC transfer timeline (both chains will act as source and receiver chains).
func (s *TransferTestSuite) TestMsgTransfer_Succeeds_Nonincentivized() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.TransferChannelOptions())
	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	chainAVersion := chainA.Config().Images[0].Version
	chainBVersion := chainB.Config().Images[0].Version

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)

		if testvalues.TotalEscrowFeatureReleases.IsSupported(chainAVersion) {
			actualTotalEscrow, err := s.QueryTotalEscrowForDenom(ctx, chainA, chainADenom)
			s.Require().NoError(err)

			expectedTotalEscrow := sdk.NewCoin(chainADenom, sdkmath.NewInt(testvalues.IBCTransferAmount))
			s.Require().Equal(expectedTotalEscrow, actualTotalEscrow)
		}
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("non-native IBC token transfer from chainB to chainA, receiver is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBIBCToken.IBCDenom()), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		s.Require().Equal(int64(0), actualBalance)

		if testvalues.TotalEscrowFeatureReleases.IsSupported(chainBVersion) {
			actualTotalEscrow, err := s.QueryTotalEscrowForDenom(ctx, chainB, chainBIBCToken.IBCDenom())
			s.Require().NoError(err)
			s.Require().Equal(sdk.NewCoin(chainBIBCToken.IBCDenom(), sdk.NewInt(0)), actualTotalEscrow) // total escrow is zero because sending chain is not source for tokens
		}
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainB, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance)
	})

	if testvalues.TotalEscrowFeatureReleases.IsSupported(chainAVersion) {
		t.Run("tokens are un-escrowed", func(t *testing.T) {
			actualTotalEscrow, err := s.QueryTotalEscrowForDenom(ctx, chainA, chainADenom)
			s.Require().NoError(err)
			s.Require().Equal(sdk.NewCoin(chainADenom, sdk.NewInt(0)), actualTotalEscrow) // total escrow is zero because tokens have come back
		})
	}
}

// TestMsgTransfer_Fails_InvalidAddress attempts to send an IBC transfer to an invalid address and ensures
// that the tokens on the sending chain are unescrowed.
func (s *TransferTestSuite) TestMsgTransfer_Fails_InvalidAddress() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.TransferChannelOptions())
	chainA, chainB := s.GetChains()

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
		s.StartRelayer(relayer)
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

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.TransferChannelOptions())
	chainA, _ := s.GetChains()
	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	chainBWalletAmount := ibc.WalletAmount{
		Address: chainBWallet.FormattedAddress(), // destination address
		Denom:   chainA.Config().Denom,
		Amount:  testvalues.IBCTransferAmount,
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
		s.StartRelayer(relayer)
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

// TestSendEnabledParam tests changing ics20 SendEnabled parameter
func (s *TransferTestSuite) TestSendEnabledParam() {
	t := s.T()
	ctx := context.TODO()

	_, channelA := s.SetupChainsRelayerAndChannel(ctx, s.TransferChannelOptions())
	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	chainAVersion := chainA.Config().Images[0].Version
	isSelfManagingParams := testvalues.SelfParamsFeatureReleases.IsSupported(chainAVersion)

	govModuleAddress, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("ensure transfer sending is enabled", func(t *testing.T) {
		enabled := s.QueryTransferParams(ctx, chainA).SendEnabled
		s.Require().True(enabled)
	})

	t.Run("ensure packets can be sent", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("change send enabled parameter to disabled", func(t *testing.T) {
		if isSelfManagingParams {
			msg := transfertypes.NewMsgUpdateParams(govModuleAddress.String(), transfertypes.NewParams(false, true))
			s.ExecuteGovProposalV1(ctx, msg, chainA, chainAWallet, 1)
		} else {
			changes := []paramsproposaltypes.ParamChange{
				paramsproposaltypes.NewParamChange(transfertypes.StoreKey, string(transfertypes.KeySendEnabled), "false"),
			}

			proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
			s.ExecuteGovProposal(ctx, chainA, chainAWallet, proposal)
		}
	})

	t.Run("ensure transfer params are disabled", func(t *testing.T) {
		enabled := s.QueryTransferParams(ctx, chainA).SendEnabled
		s.Require().False(enabled)
	})

	t.Run("ensure ics20 transfer fails", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxFailure(transferTxResp, transfertypes.ErrSendDisabled)
	})
}

// TestReceiveEnabledParam tests changing ics20 ReceiveEnabled parameter
func (s *TransferTestSuite) TestReceiveEnabledParam() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.TransferChannelOptions())
	chainA, chainB := s.GetChains()

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	var (
		chainBDenom    = chainB.Config().Denom
		chainAIBCToken = testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID) // IBC token sent to chainA

		chainAAddress = chainAWallet.FormattedAddress()
		chainBAddress = chainBWallet.FormattedAddress()
	)

	chainAVersion := chainA.Config().Images[0].Version
	isSelfManagingParams := testvalues.SelfParamsFeatureReleases.IsSupported(chainAVersion)

	govModuleAddress, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("ensure transfer receive is enabled", func(t *testing.T) {
		enabled := s.QueryTransferParams(ctx, chainA).ReceiveEnabled
		s.Require().True(enabled)
	})

	t.Run("ensure packets can be received, send from chainB to chainA", func(t *testing.T) {
		t.Run("send from chainB to chainA", func(t *testing.T) {
			transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
			s.AssertTxSuccess(transferTxResp)
		})

		t.Run("tokens are escrowed", func(t *testing.T) {
			actualBalance, err := s.GetChainBNativeBalance(ctx, chainBWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("start relayer", func(t *testing.T) {
			s.StartRelayer(relayer)
		})

		t.Run("packets are relayed", func(t *testing.T) {
			s.AssertPacketRelayed(ctx, chainA, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

			actualBalance, err := chainA.GetBalance(ctx, chainAAddress, chainAIBCToken.IBCDenom())
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("stop relayer", func(t *testing.T) {
			s.StopRelayer(ctx, relayer)
		})
	})

	t.Run("change receive enabled parameter to disabled ", func(t *testing.T) {
		if isSelfManagingParams {
			msg := transfertypes.NewMsgUpdateParams(govModuleAddress.String(), transfertypes.NewParams(false, false))
			s.ExecuteGovProposalV1(ctx, msg, chainA, chainAWallet, 1)
		} else {
			changes := []paramsproposaltypes.ParamChange{
				paramsproposaltypes.NewParamChange(transfertypes.StoreKey, string(transfertypes.KeyReceiveEnabled), "false"),
			}

			proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
			s.ExecuteGovProposal(ctx, chainA, chainAWallet, proposal)
		}
	})

	t.Run("ensure transfer params are disabled", func(t *testing.T) {
		enabled := s.QueryTransferParams(ctx, chainA).ReceiveEnabled
		s.Require().False(enabled)
	})

	t.Run("ensure ics20 transfer fails", func(t *testing.T) {
		t.Run("send from chainB to chainA", func(t *testing.T) {
			transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
			s.AssertTxSuccess(transferTxResp)
		})

		t.Run("tokens are escrowed", func(t *testing.T) {
			actualBalance, err := s.GetChainBNativeBalance(ctx, chainBWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - (testvalues.IBCTransferAmount * 2) // second send
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("start relayer", func(t *testing.T) {
			s.StartRelayer(relayer)
		})

		t.Run("tokens are unescrowed in failed acknowledgement", func(t *testing.T) {
			actualBalance, err := s.GetChainBNativeBalance(ctx, chainBWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount // only first send marked
			s.Require().Equal(expected, actualBalance)
		})
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

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.TransferChannelOptions())
	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	chainAVersion := chainA.Config().Images[0].Version
	chainBVersion := chainB.Config().Images[0].Version

	t.Run("IBC token transfer with memo from chainA to chainB", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "memo")

		if testvalues.MemoFeatureReleases.IsSupported(chainAVersion) {
			s.AssertTxSuccess(transferTxResp)
		} else {
			s.Require().Equal(uint32(2), transferTxResp.Code)
			s.Require().Contains(transferTxResp.RawLog, "errUnknownField")
		}
	})

	if !testvalues.MemoFeatureReleases.IsSupported(chainAVersion) {
		// transfer not sent, end test
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

	t.Run("packets relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		if testvalues.MemoFeatureReleases.IsSupported(chainBVersion) {
			s.Require().Equal(testvalues.IBCTransferAmount, actualBalance)
		} else {
			s.Require().Equal(int64(0), actualBalance)
		}
	})
}
