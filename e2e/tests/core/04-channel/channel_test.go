package channel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

func TestChannelUpgradeTestSuite(t *testing.T) {
	suite.Run(t, new(ChannelUpgradeTestSuite))
}

type ChannelUpgradeTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *ChannelUpgradeTestSuite) TestChannelUpgrade() {
	t := s.T()
	ctx := context.TODO()

	const upgradeVersion string = `{"fee_version":"ics29-1","app_version":"ics20-1"}`

	var (
		msgChanUpgradeInitRes channeltypes.MsgChannelUpgradeInitResponse
		msgChanUpgradeTryRes  channeltypes.MsgChannelUpgradeTryResponse
	)
	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx)

	chainA, chainB := s.GetChains()

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	t.Run("channel upgrade init", func(t *testing.T) {
		upgradeTimeout := channeltypes.NewUpgradeTimeout(clienttypes.NewHeight(0, 10000), 0)
		upgradeFields := channeltypes.NewUpgradeFields(channeltypes.UNORDERED, channelA.ConnectionHops, upgradeVersion)
		msgChanUpgradeInit := channeltypes.NewMsgChannelUpgradeInit(
			channelA.PortID, channelA.ChannelID, upgradeFields, upgradeTimeout, rlyWallet.FormattedAddress(),
		)

		s.Require().NoError(msgChanUpgradeInit.ValidateBasic())

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanUpgradeInit)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanUpgradeInitRes))

		channel, err := s.QueryChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(channeltypes.INITUPGRADE, channel.State)
	})

	t.Run("channel upgrade try", func(t *testing.T) {
		chainBChannels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainB.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Len(chainBChannels, 1)

		chainBChannel := chainBChannels[len(chainBChannels)-1]
		upgradeTimeout := channeltypes.NewUpgradeTimeout(clienttypes.NewHeight(0, 10000), 0)
		upgradeFields := channeltypes.NewUpgradeFields(channeltypes.UNORDERED, chainBChannel.ConnectionHops, upgradeVersion)

		// TODO: get channel proof
		var channelProof []byte
		// TODO: get upgrade proof
		var upgradeProof []byte

		msgChannelUpgradeTry := channeltypes.NewMsgChannelUpgradeTry(
			channelA.Counterparty.PortID,
			channelA.Counterparty.ChannelID,
			upgradeFields,
			upgradeTimeout,
			msgChanUpgradeInitRes.Upgrade,
			msgChanUpgradeInitRes.UpgradeSequence,
			channelProof,
			upgradeProof,
			clienttypes.ZeroHeight(), // proof height
			rlyWallet.FormattedAddress(),
		)

		txResp, err := s.BroadcastMessages(ctx, chainB, rlyWallet, msgChannelUpgradeTry)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanUpgradeTryRes))
		s.Require().True(msgChanUpgradeTryRes.Success)

		channel, err := s.QueryChannel(ctx, chainB, chainBChannel.PortID, chainBChannel.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(channeltypes.TRYUPGRADE, channel.State)
	})
}
