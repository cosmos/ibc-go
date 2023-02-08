package interchain_accounts

import (
	"context"
	"testing"

	controllertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	test "github.com/strangelove-ventures/interchaintest/v7/testutil"
)

func (s *InterchainAccountsTestSuite) TestInterchainAccounts_Localhost() {
	t := s.T()
	ctx := context.TODO()

	_, _ = s.SetupChainsRelayerAndChannel(ctx)
	chainA, _ := s.GetChains()

	// chainADenom := chainA.Config().Denom

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	// userBWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	var (
		msgChanOpenInitRes channeltypes.MsgChannelOpenInitResponse
		msgChanOpenTryRes  channeltypes.MsgChannelOpenTryResponse
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA), "failed to wait for blocks")

	version := icatypes.NewDefaultMetadataString(connectiontypes.LocalhostID, connectiontypes.LocalhostID)
	controllerPortID, err := icatypes.NewControllerPortID(userAWallet.FormattedAddress())
	s.Require().NoError(err)

	t.Run("channel open init localhost - broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(connectiontypes.LocalhostID, userAWallet.FormattedAddress(), version)

		txResp, err := s.BroadcastMessages(ctx, chainA, userAWallet, msgRegisterAccount)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenInitRes))
	})

	t.Run("channel open try localhost", func(t *testing.T) {
		msgChanOpenTry := channeltypes.NewMsgChannelOpenTry(
			icatypes.HostPortID, icatypes.Version,
			channeltypes.ORDERED, []string{connectiontypes.LocalhostID},
			controllerPortID, msgChanOpenInitRes.GetChannelId(),
			version, nil, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenTry)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenTryRes))
	})

	t.Run("channel open ack localhost", func(t *testing.T) {
		msgChanOpenAck := channeltypes.NewMsgChannelOpenAck(
			controllerPortID, msgChanOpenInitRes.GetChannelId(),
			msgChanOpenTryRes.GetChannelId(), msgChanOpenTryRes.GetVersion(),
			nil, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenAck)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})

	t.Run("channel open confirm localhost", func(t *testing.T) {
		msgChanOpenConfirm := channeltypes.NewMsgChannelOpenConfirm(
			icatypes.HostPortID, msgChanOpenTryRes.GetChannelId(),
			nil, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenConfirm)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})

	t.Run("query localhost interchain accounts channel ends", func(t *testing.T) {
		channelEndA, err := s.QueryChannel(ctx, chainA, controllerPortID, msgChanOpenInitRes.GetChannelId())
		s.Require().NoError(err)
		s.Require().NotNil(channelEndA)

		t.Logf("controller channel end: %v", channelEndA)

		channelEndB, err := s.QueryChannel(ctx, chainA, icatypes.HostPortID, msgChanOpenTryRes.GetChannelId())
		s.Require().NoError(err)
		s.Require().NotNil(channelEndB)

		t.Logf("host channel end: %v", channelEndB)

		s.Require().Equal(channelEndA.GetConnectionHops(), channelEndB.GetConnectionHops())
	})
}
