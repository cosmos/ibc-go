//go:build !test_e2e

package transfer

import (
	"context"
	"testing"

	test "github.com/strangelove-ventures/interchaintest/v8/testutil"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	localhost "github.com/cosmos/ibc-go/v8/modules/light-clients/09-localhost"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

// TestMsgTransfer_Localhost creates two wallets on a single chain and performs MsgTransfers back and forth
// to ensure ibc functions as expected on localhost. This test is largely the same as TestMsgTransfer_Succeeds_Nonincentivized
// except that chain B is replaced with an additional wallet on chainA.
func (s *TransferTestSuite) TestMsgTransfer_Localhost() {
	t := s.T()

	ctx := context.TODO()

	_, err := s.rly.GetChannels(ctx, s.GetRelayerExecReporter(), s.chainA.Config().ChainID)
	s.Require().NoError(err)

	chainADenom := s.chainA.Config().Denom

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, s.chainA)
	userAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, s.chainA)
	userBWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, s.chainA)

	var (
		msgChanOpenInitRes channeltypes.MsgChannelOpenInitResponse
		msgChanOpenTryRes  channeltypes.MsgChannelOpenTryResponse
		ack                []byte
		packet             channeltypes.Packet
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, s.chainA), "failed to wait for blocks")

	t.Run("verify begin blocker was executed", func(t *testing.T) {
		cs, err := s.QueryClientState(ctx, s.chainA, exported.LocalhostClientID)
		s.Require().NoError(err)
		originalHeight := cs.GetLatestHeight()

		s.Require().NoError(test.WaitForBlocks(ctx, 1, s.chainA), "failed to wait for blocks")

		cs, err = s.QueryClientState(ctx, s.chainA, exported.LocalhostClientID)
		s.Require().NoError(err)
		s.Require().True(cs.GetLatestHeight().GT(originalHeight), "client state height was not incremented")
	})

	t.Run("channel open init localhost", func(t *testing.T) {
		msgChanOpenInit := channeltypes.NewMsgChannelOpenInit(
			transfertypes.PortID, transfertypes.Version,
			channeltypes.UNORDERED, []string{exported.LocalhostConnectionID},
			transfertypes.PortID, rlyWallet.FormattedAddress(),
		)

		s.Require().NoError(msgChanOpenInit.ValidateBasic())

		txResp := s.BroadcastMessages(ctx, s.chainA, rlyWallet, s.chainB, msgChanOpenInit)
		s.AssertTxSuccess(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenInitRes))
	})

	t.Run("channel open try localhost", func(t *testing.T) {
		msgChanOpenTry := channeltypes.NewMsgChannelOpenTry(
			transfertypes.PortID, transfertypes.Version,
			channeltypes.UNORDERED, []string{exported.LocalhostConnectionID},
			transfertypes.PortID, msgChanOpenInitRes.ChannelId,
			transfertypes.Version, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, s.chainA, rlyWallet, s.chainB, msgChanOpenTry)
		s.AssertTxSuccess(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenTryRes))
	})

	t.Run("channel open ack localhost", func(t *testing.T) {
		msgChanOpenAck := channeltypes.NewMsgChannelOpenAck(
			transfertypes.PortID, msgChanOpenInitRes.ChannelId,
			msgChanOpenTryRes.ChannelId, transfertypes.Version,
			localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, s.chainA, rlyWallet, s.chainB, msgChanOpenAck)
		s.AssertTxSuccess(txResp)
	})

	t.Run("channel open confirm localhost", func(t *testing.T) {
		msgChanOpenConfirm := channeltypes.NewMsgChannelOpenConfirm(
			transfertypes.PortID, msgChanOpenTryRes.ChannelId,
			localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, s.chainA, rlyWallet, s.chainA, msgChanOpenConfirm)
		s.AssertTxSuccess(txResp)
	})

	t.Run("query localhost transfer channel ends", func(t *testing.T) {
		channelEndA, err := s.QueryChannel(ctx, s.chainA, transfertypes.PortID, msgChanOpenInitRes.ChannelId)
		s.Require().NoError(err)
		s.Require().NotNil(channelEndA)

		channelEndB, err := s.QueryChannel(ctx, s.chainA, transfertypes.PortID, msgChanOpenTryRes.ChannelId)
		s.Require().NoError(err)
		s.Require().NotNil(channelEndB)

		s.Require().Equal(channelEndA.GetConnectionHops(), channelEndB.GetConnectionHops())
	})

	t.Run("send packet localhost ibc transfer", func(t *testing.T) {
		var err error
		txResp := s.Transfer(ctx, s.chainA, userAWallet, transfertypes.PortID, msgChanOpenInitRes.ChannelId, testvalues.DefaultTransferAmount(chainADenom), userAWallet.FormattedAddress(), userBWallet.FormattedAddress(), clienttypes.NewHeight(1, 100), 0, "", s.chainB)
		s.AssertTxSuccess(txResp)

		packet, err = ibctesting.ParsePacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, userAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("recv packet localhost ibc transfer", func(t *testing.T) {
		var err error
		msgRecvPacket := channeltypes.NewMsgRecvPacket(packet, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, s.chainA, rlyWallet, s.chainB, msgRecvPacket)
		s.AssertTxSuccess(txResp)

		ack, err = ibctesting.ParseAckFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(ack)
	})

	t.Run("acknowledge packet localhost ibc transfer", func(t *testing.T) {
		msgAcknowledgement := channeltypes.NewMsgAcknowledgement(packet, ack, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, s.chainA, rlyWallet, s.chainB, msgAcknowledgement)
		s.AssertTxSuccess(txResp)
	})

	t.Run("verify tokens transferred", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, s.chainA, transfertypes.PortID, msgChanOpenInitRes.ChannelId, 1)

		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, msgChanOpenTryRes.ChannelId)
		actualBalance, err := s.QueryBalance(ctx, s.chainA, userBWallet.FormattedAddress(), ibcToken.IBCDenom())

		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})
}
