package transfer

import (
	"context"
	"strconv"
	"testing"
	"time"

	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/strangelove-ventures/ibctest/v6/ibc"
	"github.com/strangelove-ventures/ibctest/v6/test"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
)

func TestTransferTestSuite(t *testing.T) {
	suite.Run(t, new(TransferTestSuite))
}

type TransferTestSuite struct {
	testsuite.E2ETestSuite
}

// QueryTransferSendEnabledParam queries the on-chain send enabled param for the transfer module
func (s *TransferTestSuite) QueryTransferSendEnabledParam(ctx context.Context, chain ibc.Chain) bool {
	queryClient := s.GetChainGRCPClients(chain).ParamsQueryClient
	res, err := queryClient.Params(ctx, &paramsproposaltypes.QueryParamsRequest{
		Subspace: transfertypes.StoreKey,
		Key:      string(transfertypes.KeySendEnabled),
	})
	s.Require().NoError(err)

	enabled, err := strconv.ParseBool(res.Param.Value)
	s.Require().NoError(err)

	return enabled
}

// QueryTransferSendEnabledParam queries the on-chain receive enabled param for the transfer module
func (s *TransferTestSuite) QueryTransferReceiveEnabledParam(ctx context.Context, chain ibc.Chain) bool {
	queryClient := s.GetChainGRCPClients(chain).ParamsQueryClient
	res, err := queryClient.Params(ctx, &paramsproposaltypes.QueryParamsRequest{
		Subspace: transfertypes.StoreKey,
		Key:      string(transfertypes.KeyReceiveEnabled),
	})
	s.Require().NoError(err)

	enabled, err := strconv.ParseBool(res.Param.Value)
	s.Require().NoError(err)

	return enabled
}

// TestMsgTransfer_Succeeds_Nonincentivized will test sending successful IBC transfers from chainA to chainB.
// The transfer will occur over a basic transfer channel (non incentivized) and both native and non-native tokens
// will be sent forwards and backwards in the IBC transfer timeline (both chains will act as source and receiver chains).
func (s *TransferTestSuite) TestMsgTransfer_Succeeds_Nonincentivized() {
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

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp, err := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
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

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("non-native IBC token transfer from chainB to chainA, receiver is source of tokens", func(t *testing.T) {
		transferTxResp, err := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBIBCToken.IBCDenom()), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.Require().NoError(err)
		s.AssertValidTxResponse(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		s.Require().Equal(int64(0), actualBalance)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainB, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance)
	})
}

// TestMsgTransfer_Fails_InvalidAddress attempts to send an IBC transfer to an invalid address and ensures
// that the tokens on the sending chain are unescrowed.
func (s *TransferTestSuite) TestMsgTransfer_Fails_InvalidAddress() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("native IBC token transfer from chainA to invalid address", func(t *testing.T) {
		transferTxResp, err := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, testvalues.InvalidAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
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

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, chainB := s.GetChains()
	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	chainBWalletAmount := ibc.WalletAmount{
		Address: chainBWallet.Bech32Address(chainB.Config().Bech32Prefix), // destination address
		Denom:   chainA.Config().Denom,
		Amount:  testvalues.IBCTransferAmount,
	}

	t.Run("IBC transfer packet timesout", func(t *testing.T) {
		tx, err := chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName, chainBWalletAmount, testvalues.ImmediatelyTimeout())
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

	_, channelA := s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.Bech32Address(chainB.Config().Bech32Prefix)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("ensure transfer sending is enabled", func(t *testing.T) {
		enabled := s.QueryTransferSendEnabledParam(ctx, chainA)
		s.Require().True(enabled)
	})

	t.Run("ensure packets can be sent", func(t *testing.T) {
		transferTxResp, err := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.Require().NoError(err)
		s.AssertValidTxResponse(transferTxResp)
	})

	t.Run("change send enabled parameter to disabled", func(t *testing.T) {
		changes := []paramsproposaltypes.ParamChange{
			paramsproposaltypes.NewParamChange(transfertypes.StoreKey, string(transfertypes.KeySendEnabled), "false"),
		}

		proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
		s.ExecuteGovProposal(ctx, chainA, chainAWallet, proposal)
	})

	t.Run("ensure transfer params are disabled", func(t *testing.T) {
		enabled := s.QueryTransferSendEnabledParam(ctx, chainA)
		s.Require().False(enabled)
	})

	t.Run("ensure ics20 transfer fails", func(t *testing.T) {
		transferTxResp, err := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.Require().NoError(err)
		s.Require().Equal(transfertypes.ErrSendDisabled.ABCICode(), transferTxResp.Code)
	})
}

// TestReceiveEnabledParam tests changing ics20 ReceiveEnabled parameter
func (s *TransferTestSuite) TestReceiveEnabledParam() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, chainB := s.GetChains()

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	var (
		chainBDenom    = chainB.Config().Denom
		chainAIBCToken = testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID) // IBC token sent to chainA

		chainAAddress = chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)
		chainBAddress = chainBWallet.Bech32Address(chainB.Config().Bech32Prefix)
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("ensure transfer receive is enabled", func(t *testing.T) {
		enabled := s.QueryTransferReceiveEnabledParam(ctx, chainA)
		s.Require().True(enabled)
	})

	t.Run("ensure packets can be received, send from chainB to chainA", func(t *testing.T) {
		t.Run("send from chainB to chainA", func(t *testing.T) {
			transferTxResp, err := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
			s.Require().NoError(err)
			s.AssertValidTxResponse(transferTxResp)
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

	t.Run("change send enabled parameter to disabled ", func(t *testing.T) {
		changes := []paramsproposaltypes.ParamChange{
			paramsproposaltypes.NewParamChange(transfertypes.StoreKey, string(transfertypes.KeyReceiveEnabled), "false"),
		}

		proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
		s.ExecuteGovProposal(ctx, chainA, chainAWallet, proposal)
	})

	t.Run("ensure transfer params are disabled", func(t *testing.T) {
		enabled := s.QueryTransferReceiveEnabledParam(ctx, chainA)
		s.Require().False(enabled)
	})

	t.Run("ensure ics20 transfer fails", func(t *testing.T) {
		t.Run("send from chainB to chainA", func(t *testing.T) {
			transferTxResp, err := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
			s.Require().NoError(err)
			s.AssertValidTxResponse(transferTxResp)
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

// memoFeatureReleases represents the releases the memo field was released in.
var memoFeatureReleases = testsuite.FeatureReleases{
	MajorVersion: "v6",
	MinorVersions: []string{
		"v2.5",
		"v3.4",
		"v4.2",
		"v5.1",
	},
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

	t.Run("IBC token transfer with memo from chainA to chainB", func(t *testing.T) {
		transferTxResp, err := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "memo")
		s.Require().NoError(err)

		if memoFeatureReleases.IsSupported(chainAVersion) {
			s.AssertValidTxResponse(transferTxResp)
		} else {
			s.Require().Equal(uint32(2), transferTxResp.Code)
			s.Require().Contains(transferTxResp.RawLog, "errUnknownField")
		}
	})

	if !memoFeatureReleases.IsSupported(chainAVersion) {
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

		if memoFeatureReleases.IsSupported(chainBVersion) {
			s.Require().Equal(testvalues.IBCTransferAmount, actualBalance)
		} else {
			s.Require().Equal(int64(0), actualBalance)
		}
	})
}

// transferChannelOptions configures both of the chains to have non-incentivized transfer channels.
func transferChannelOptions() func(options *ibc.CreateChannelOptions) {
	return func(opts *ibc.CreateChannelOptions) {
		opts.Version = transfertypes.Version
		opts.SourcePortName = transfertypes.PortID
		opts.DestPortName = transfertypes.PortID
	}
}
