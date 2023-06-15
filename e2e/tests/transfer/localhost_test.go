package transfer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	localhost "github.com/cosmos/ibc-go/v7/modules/light-clients/09-localhost"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	test "github.com/strangelove-ventures/interchaintest/v7/testutil"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
)

func TestTransferLocalhostTestSuite(t *testing.T) {
	suite.Run(t, new(LocalhostTransferTestSuite))
}

type LocalhostTransferTestSuite struct {
	testsuite.E2ETestSuite
}

// TestMsgTransfer_Localhost creates two wallets on a single chain and performs MsgTransfers back and forth
// to ensure ibc functions as expected on localhost. This test is largely the same as TestMsgTransfer_Succeeds_Nonincentivized
// except that chain B is replaced with an additional wallet on chainA.
func (s *LocalhostTransferTestSuite) TestMsgTransfer_Localhost() {
	t := s.T()
	ctx := context.TODO()

	_, _ = s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, _ := s.GetChains()

	chainADenom := chainA.Config().Denom

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userBWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	var (
		msgChanOpenInitRes channeltypes.MsgChannelOpenInitResponse
		msgChanOpenTryRes  channeltypes.MsgChannelOpenTryResponse
		ack                []byte
		packet             channeltypes.Packet
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA), "failed to wait for blocks")

	t.Run("channel open init localhost", func(t *testing.T) {
		msgChanOpenInit := channeltypes.NewMsgChannelOpenInit(
			transfertypes.PortID, transfertypes.Version,
			channeltypes.UNORDERED, []string{exported.LocalhostConnectionID},
			transfertypes.PortID, rlyWallet.FormattedAddress(),
		)

		s.Require().NoError(msgChanOpenInit.ValidateBasic())

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenInit)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenInitRes))
	})

	t.Run("channel open try localhost", func(t *testing.T) {
		msgChanOpenTry := channeltypes.NewMsgChannelOpenTry(
			transfertypes.PortID, transfertypes.Version,
			channeltypes.UNORDERED, []string{exported.LocalhostConnectionID},
			transfertypes.PortID, msgChanOpenInitRes.GetChannelId,
			transfertypes.Version, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenTry)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenTryRes))
	})

	t.Run("channel open ack localhost", func(t *testing.T) {
		msgChanOpenAck := channeltypes.NewMsgChannelOpenAck(
			transfertypes.PortID, msgChanOpenInitRes.GetChannelId,
			msgChanOpenTryRes.ChannelId, transfertypes.Version,
			localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenAck)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})

	t.Run("channel open confirm localhost", func(t *testing.T) {
		msgChanOpenConfirm := channeltypes.NewMsgChannelOpenConfirm(
			transfertypes.PortID, msgChanOpenTryRes.ChannelId,
			localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenConfirm)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})

	t.Run("query localhost transfer channel ends", func(t *testing.T) {
		channelEndA, err := s.QueryChannel(ctx, chainA, transfertypes.PortID, msgChanOpenInitRes.GetChannelId)
		s.Require().NoError(err)
		s.Require().NotNil(channelEndA)

		channelEndB, err := s.QueryChannel(ctx, chainA, transfertypes.PortID, msgChanOpenTryRes.ChannelId)
		s.Require().NoError(err)
		s.Require().NotNil(channelEndB)

		s.Require().Equal(channelEndA.GetConnectionHops(), channelEndB.GetConnectionHops())
	})

	t.Run("send packet localhost ibc transfer", func(t *testing.T) {
		txResp, err := s.Transfer(ctx, chainA, userAWallet, transfertypes.PortID, msgChanOpenInitRes.GetChannelId, testvalues.DefaultTransferAmount(chainADenom), userAWallet.FormattedAddress(), userBWallet.FormattedAddress(), clienttypes.NewHeight(1, 100), 0, "")
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)

		events := testsuite.ABCIToSDKEvents(txResp.Events)
		packet, err = ibctesting.ParsePacketFromEvents(events)
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
		msgRecvPacket := channeltypes.NewMsgRecvPacket(packet, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgRecvPacket)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)

		events := testsuite.ABCIToSDKEvents(txResp.Events)
		ack, err = ibctesting.ParseAckFromEvents(events)
		s.Require().NoError(err)
		s.Require().NotNil(ack)
	})

	t.Run("acknowledge packet localhost ibc transfer", func(t *testing.T) {
		msgAcknowledgement := channeltypes.NewMsgAcknowledgement(packet, ack, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgAcknowledgement)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})

	t.Run("verify tokens transferred", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, transfertypes.PortID, msgChanOpenInitRes.GetChannelId, 1)

		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, msgChanOpenTryRes.ChannelId)
		actualBalance, err := chainA.GetBalance(ctx, userBWallet.FormattedAddress(), ibcToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})
}
