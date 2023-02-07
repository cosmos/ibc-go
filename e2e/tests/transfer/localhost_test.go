package transfer

import (
	"context"
	"testing"

	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"

	"github.com/cosmos/ibc-go/e2e/testvalues"
	test "github.com/strangelove-ventures/interchaintest/v7/testutil"
)

// TestMsgTransfer_Localhost creates two wallets on a single chain and performs MsgTransfers back and forth
// to ensure ibc functions as expected on localhost. This test is largely the same as TestMsgTransfer_Succeeds_Nonincentivized
// except that chain B is replaced with an additional wallet on chainA.
func (s *TransferTestSuite) TestMsgTransfer_Localhost() {
	t := s.T()
	ctx := context.TODO()

	_, _ = s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, _ := s.GetChains()

	// chainADenom := chainA.Config().Denom
	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA), "failed to wait for blocks")

	t.Run("channel open init localhost", func(t *testing.T) {
		msgChannelOpenInit := channeltypes.NewMsgChannelOpenInit(
			transfertypes.PortID, transfertypes.Version,
			channeltypes.UNORDERED, []string{connectiontypes.LocalhostID},
			transfertypes.PortID, rlyWallet.FormattedAddress(),
		)

		s.Require().NoError(msgChannelOpenInit.ValidateBasic())

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChannelOpenInit)
		s.AssertValidTxResponse(txResp)
		s.Require().NoError(err)

		s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA), "failed to wait for blocks")
	})

	t.Run("channel open try localhost", func(t *testing.T) {
		msgChannelOpenTry := channeltypes.NewMsgChannelOpenTry(
			transfertypes.PortID, transfertypes.Version,
			channeltypes.UNORDERED, []string{connectiontypes.LocalhostID},
			transfertypes.PortID, "channel-1", // TODO: parse channel ID from response
			transfertypes.Version, nil, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		_, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChannelOpenTry)
		s.Require().NoError(err)
		// s.AssertValidTxResponse(txResp)

		s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA), "failed to wait for blocks")
	})

	t.Run("channel open ack localhost", func(t *testing.T) {
		msgChannelOpenAck := channeltypes.NewMsgChannelOpenAck(
			transfertypes.PortID, "channel-1", // TODO: Parse channel ID from response
			"channel-2", transfertypes.Version,
			nil, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		_, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChannelOpenAck)
		s.Require().NoError(err)
		// s.AssertValidTxResponse(txResp)

		s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA), "failed to wait for blocks")
	})

	t.Run("channel open confirm localhost", func(t *testing.T) {
		msgChannelOpenConfirm := channeltypes.NewMsgChannelOpenConfirm(
			transfertypes.PortID, "channel-2",
			nil, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		_, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChannelOpenConfirm)
		s.Require().NoError(err)
		// s.AssertValidTxResponse(txResp)

		s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA), "failed to wait for blocks")
	})

	t.Run("query localhost transfer channel", func(t *testing.T) {
		channel, err := s.QueryChannel(ctx, chainA, transfertypes.PortID, "channel-1")
		s.Require().NoError(err)
		s.Require().NotNil(channel)

		t.Logf("output channel response: %v", channel)
	})

	// t.Run("localhost IBC token transfer", func(t *testing.T) {
	// 	transferTxResp, err := s.Transfer(ctx, chainA, chainAWallet1, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAWallet1.FormattedAddress(), chainAWallet2.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
	// 	s.Require().NoError(err)
	// 	s.AssertValidTxResponse(transferTxResp)
	// })

	// t.Run("tokens are escrowed", func(t *testing.T) {
	// 	actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet1)
	// 	s.Require().NoError(err)

	// 	expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
	// 	s.Require().Equal(expected, actualBalance)
	// })

	// t.Run("start relayer", func(t *testing.T) {
	// 	s.StartRelayer(relayer)
	// })

	// ibcToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	// t.Run("packets are relayed", func(t *testing.T) {
	// 	s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

	// 	actualBalance, err := chainA.GetBalance(ctx, chainAWallet2.FormattedAddress(), ibcToken.IBCDenom())
	// 	s.Require().NoError(err)

	// 	expected := testvalues.IBCTransferAmount
	// 	s.Require().Equal(expected, actualBalance)
	// })

	// t.Run("non-native IBC token transfer using localhost receiver is source of tokens", func(t *testing.T) {
	// 	transferTxResp, err := s.Transfer(ctx, chainA, chainAWallet2, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(ibcToken.IBCDenom()), chainAWallet2.FormattedAddress(), chainAWallet1.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
	// 	s.Require().NoError(err)
	// 	s.AssertValidTxResponse(transferTxResp)
	// })

	// t.Run("tokens are escrowed", func(t *testing.T) {
	// 	actualBalance, err := chainA.GetBalance(ctx, chainAWallet2.FormattedAddress(), ibcToken.IBCDenom())
	// 	s.Require().NoError(err)

	// 	s.Require().Equal(int64(0), actualBalance)
	// })

	// s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA), "failed to wait for blocks")

	// t.Run("packets are relayed", func(t *testing.T) {
	// 	s.AssertPacketRelayed(ctx, chainA, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

	// 	actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet1)
	// 	s.Require().NoError(err)

	// 	expected := testvalues.StartingTokenAmount
	// 	s.Require().Equal(expected, actualBalance)
	// })
}
