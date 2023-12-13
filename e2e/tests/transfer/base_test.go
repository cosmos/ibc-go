//go:build !test_e2e

package transfer

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func TestTransferTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferTestSuite))
}

type TransferTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *TransferTestSuite) SetupTest() {
	ctx := context.TODO()
	chainA, chainB := s.GetChains()
	relayer := s.SetupRelayer(ctx, s.TransferChannelOptions(), chainA, chainB)
	s.SetChainsIntoSuite(chainA, chainB)
	s.SetRelayerIntoSuite(relayer)
}

// QueryTransferSendEnabledParam queries the on-chain send enabled param for the transfer module
func (s *TransferTestSuite) QueryTransferParams(ctx context.Context, chain ibc.Chain) transfertypes.Params {
	queryClient := s.GetChainGRCPClients(chain).TransferQueryClient
	res, err := queryClient.Params(ctx, &transfertypes.QueryParamsRequest{})
	s.Require().NoError(err)
	return *res.Params
}

// TestMsgTransfer_Fails_InvalidAddress attempts to send an IBC transfer to an invalid address and ensures
// that the tokens on the sending chain are unescrowed.
func (s *TransferTestSuite) TestMsgTransfer_Fails_InvalidAddress() {
	t := s.T()

	ctx := context.TODO()

	chainA, chainB := s.GetChains()
	relayer := s.GetRelayerFromSuite()

	channelA, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
	s.Require().NoError(err)
	chainAChannels := channelA[len(channelA)-1]

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, chainA)
	chainAAddress := chainAWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("native IBC token transfer from chainA to invalid address", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, chainAChannels.PortID, chainAChannels.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, testvalues.InvalidAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", chainB)
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
		s.AssertPacketRelayed(ctx, chainA, chainAChannels.PortID, chainAChannels.ChannelID, 1)
	})

	t.Run("token transfer amount unescrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance)
	})
}

// TestSendEnabledParam tests changing ics20 SendEnabled parameter
func (s *TransferTestSuite) TestSendEnabledParam() {
	t := s.T()

	ctx := context.TODO()

	chainA, chainB := s.GetChains()
	relayer := s.GetRelayerFromSuite()

	channelA, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
	s.Require().NoError(err)
	chainAChannels := channelA[len(channelA)-1]

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, chainA)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount, chainB)
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
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, chainAChannels.PortID, chainAChannels.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", chainB)
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("change send enabled parameter to disabled", func(t *testing.T) {
		if isSelfManagingParams {
			msg := transfertypes.NewMsgUpdateParams(govModuleAddress.String(), transfertypes.NewParams(false, true))
			s.ExecuteAndPassGovV1Proposal(ctx, msg, chainA, chainAWallet)
		} else {
			changes := []paramsproposaltypes.ParamChange{
				paramsproposaltypes.NewParamChange(transfertypes.StoreKey, string(transfertypes.KeySendEnabled), "false"),
			}

			proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
			s.ExecuteAndPassGovV1Beta1Proposal(ctx, chainA, chainAWallet, proposal, chainB)
		}
	})

	t.Run("ensure transfer params are disabled", func(t *testing.T) {
		enabled := s.QueryTransferParams(ctx, chainA).SendEnabled
		s.Require().False(enabled)
	})

	t.Run("ensure ics20 transfer fails", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, chainAChannels.PortID, chainAChannels.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", chainB)
		s.AssertTxFailure(transferTxResp, transfertypes.ErrSendDisabled)
	})
}

// TestReceiveEnabledParam tests changing ics20 ReceiveEnabled parameter
func (s *TransferTestSuite) TestReceiveEnabledParam() {
	t := s.T()
	t.Parallel()
	ctx := context.TODO()

	chainA, chainB := s.GetChains()
	relayer := s.GetRelayerFromSuite()

	channelA, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
	s.Require().NoError(err)
	chainAChannels := channelA[len(channelA)-1]

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, chainA)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount, chainB)

	var (
		chainBDenom    = chainB.Config().Denom
		chainAIBCToken = testsuite.GetIBCToken(chainBDenom, chainAChannels.PortID, chainAChannels.ChannelID) // IBC token sent to chainA

		chainAAddress = chainAWallet.FormattedAddress()
		chainBAddress = chainBWallet.FormattedAddress()
	)

	chainAVersion := chainA.Config().Images[0].Version
	isSelfManagingParams := testvalues.SelfParamsFeatureReleases.IsSupported(chainAVersion)

	s.InitGRPCClients(chainA)
	s.InitGRPCClients(chainB)
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
			transferTxResp := s.Transfer(ctx, chainB, chainBWallet, chainAChannels.Counterparty.PortID, chainAChannels.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "", chainA)
			s.AssertTxSuccess(transferTxResp)
		})

		t.Run("tokens are escrowed", func(t *testing.T) {
			actualBalance, err := s.GetChainBNativeBalance(ctx, chainBWallet, chainB)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("start relayer", func(t *testing.T) {
			s.StartRelayer(relayer)
		})

		t.Run("packets are relayed", func(t *testing.T) {
			s.AssertPacketRelayed(ctx, chainA, chainAChannels.Counterparty.PortID, chainAChannels.Counterparty.ChannelID, 1)
			actualBalance, err := s.QueryBalance(ctx, chainA, chainAAddress, chainAIBCToken.IBCDenom())

			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance.Int64())
		})

		t.Run("stop relayer", func(t *testing.T) {
			s.StopRelayer(ctx, relayer)
		})
	})

	t.Run("change receive enabled parameter to disabled ", func(t *testing.T) {
		if isSelfManagingParams {
			msg := transfertypes.NewMsgUpdateParams(govModuleAddress.String(), transfertypes.NewParams(false, false))
			s.ExecuteAndPassGovV1Proposal(ctx, msg, chainA, chainAWallet)
		} else {
			changes := []paramsproposaltypes.ParamChange{
				paramsproposaltypes.NewParamChange(transfertypes.StoreKey, string(transfertypes.KeyReceiveEnabled), "false"),
			}

			proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
			s.ExecuteAndPassGovV1Beta1Proposal(ctx, chainA, chainAWallet, proposal, chainB)
		}
	})

	t.Run("ensure transfer params are disabled", func(t *testing.T) {
		enabled := s.QueryTransferParams(ctx, chainA).ReceiveEnabled
		s.Require().False(enabled)
	})

	t.Run("ensure ics20 transfer fails", func(t *testing.T) {
		t.Run("send from chainB to chainA", func(t *testing.T) {
			transferTxResp := s.Transfer(ctx, chainB, chainBWallet, chainAChannels.Counterparty.PortID, chainAChannels.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "", chainA)
			s.AssertTxSuccess(transferTxResp)
		})

		t.Run("tokens are escrowed", func(t *testing.T) {
			actualBalance, err := s.GetChainBNativeBalance(ctx, chainBWallet, chainB)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - (testvalues.IBCTransferAmount * 2) // second send
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("start relayer", func(t *testing.T) {
			s.StartRelayer(relayer)
		})

		t.Run("tokens are unescrowed in failed acknowledgement", func(t *testing.T) {
			actualBalance, err := s.GetChainBNativeBalance(ctx, chainBWallet, chainB)
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

	chainA, chainB := s.GetChains()
	relayer := s.GetRelayerFromSuite()

	channelA, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
	s.Require().NoError(err)
	chainAChannels := channelA[len(channelA)-1]

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, chainA)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount, chainB)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("IBC token transfer with memo from chainA to chainB", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, chainAChannels.PortID, chainAChannels.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "memo", chainB)
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

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, chainAChannels.Counterparty.PortID, chainAChannels.Counterparty.ChannelID)

	t.Run("packets relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, chainAChannels.PortID, chainAChannels.ChannelID, 1)
		actualBalance, err := s.QueryBalance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())

		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, actualBalance.Int64())
	})
}
