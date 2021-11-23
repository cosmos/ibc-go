package keeper_test

import (
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

func (suite *KeeperTestSuite) TestInitGenesis() {
	suite.SetupTest()

	refundAcc := suite.chainA.SenderAccount.GetAddress()
	ackFee := validCoins
	receiveFee := validCoins2
	timeoutFee := validCoins3
	packetId := &channeltypes.PacketId{ChannelId: ibctesting.FirstChannelID, PortId: types.PortID, Sequence: uint64(1)}
	fee := types.Fee{ackFee, receiveFee, timeoutFee}

	genesisState := types.GenesisState{
		IdentifiedFees: []*types.IdentifiedPacketFee{
			{
				PacketId:      packetId,
				Fee:           fee,
				RefundAddress: refundAcc.String(),
				Relayers:      nil,
			},
		},
	}

	suite.chainA.GetSimApp().IBCFeeKeeper.InitGenesis(suite.chainA.GetContext(), genesisState)

	identifiedFee, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeInEscrow(suite.chainA.GetContext(), packetId)
	suite.Require().True(found)
	suite.Require().Equal(genesisState.IdentifiedFees[0], &identifiedFee)
}

/*
func (suite *InterchainAccountsTestSuite) TestExportGenesis() {
	suite.SetupTest()
	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	suite.Require().NoError(err)

	genesisState := ica.ExportGenesis(suite.chainA.GetContext(), suite.chainA.GetSimApp().ICAKeeper)

	suite.Require().Equal([]string{types.PortID, TestPortID}, genesisState.GetPorts())

	suite.Require().Equal(path.EndpointA.ChannelID, genesisState.ActiveChannels[0].ChannelId)
	suite.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.ActiveChannels[0].PortId)

	suite.Require().Equal(TestAccAddress.String(), genesisState.InterchainAccounts[0].AccountAddress)
	suite.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.InterchainAccounts[0].PortId)
}
*/
