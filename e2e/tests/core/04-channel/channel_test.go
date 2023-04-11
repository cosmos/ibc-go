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

	_, channelA := s.SetupChainsRelayerAndChannel(ctx)

	chainA, _ := s.GetChains()

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	counterParty := channeltypes.NewCounterparty(channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)
	proposedUpgradeChannel := channeltypes.NewChannel(channeltypes.INITUPGRADE, channeltypes.UNORDERED, counterParty, channelA.ConnectionHops, `{"fee_version":"ics29-1","app_version":"ics20-1"}`)

	t.Run("channel upgrade init", func(t *testing.T) {
		msgChanUpgradeInit := channeltypes.NewMsgChannelUpgradeInit(
			channelA.PortID, channelA.ChannelID, proposedUpgradeChannel, clienttypes.NewHeight(0, 10000), 0, rlyWallet.FormattedAddress(),
		)

		s.Require().NoError(msgChanUpgradeInit.ValidateBasic())

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanUpgradeInit)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})
}
