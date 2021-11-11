package keeper_test

import (
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v2/testing"
)

func (suite *KeeperTestSuite) TestInitGenesis() {
	suite.SetupTest()

	genesisState := types.ControllerGenesisState{
		ActiveChannels: []*types.ActiveChannel{
			{
				PortId:    TestPortID,
				ChannelId: ibctesting.FirstChannelID,
			},
		},
		InterchainAccounts: []*types.RegisteredInterchainAccount{
			{
				PortId:         TestPortID,
				AccountAddress: TestAccAddress.String(),
			},
		},
		Ports: []string{TestPortID},
	}

	keeper.InitGenesis(suite.chainA.GetContext(), suite.chainA.GetSimApp().ICAControllerKeeper, genesisState)

	channelID, found := suite.chainA.GetSimApp().ICAControllerKeeper.GetActiveChannelID(suite.chainA.GetContext(), TestPortID)
	suite.Require().True(found)
	suite.Require().Equal(ibctesting.FirstChannelID, channelID)

	accountAdrr, found := suite.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(suite.chainA.GetContext(), TestPortID)
	suite.Require().True(found)
	suite.Require().Equal(TestAccAddress.String(), accountAdrr)
}

func (suite *KeeperTestSuite) TestExportGenesis() {
	suite.SetupTest()

	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	suite.Require().NoError(err)

	genesisState := keeper.ExportGenesis(suite.chainA.GetContext(), suite.chainA.GetSimApp().ICAControllerKeeper)

	suite.Require().Equal(path.EndpointA.ChannelID, genesisState.ActiveChannels[0].ChannelId)
	suite.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.ActiveChannels[0].PortId)

	suite.Require().Equal(TestAccAddress.String(), genesisState.InterchainAccounts[0].AccountAddress)
	suite.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.InterchainAccounts[0].PortId)

	// TODO: Is there a bug here? - Answer: Yes Solution: Split the store keys
	// GetAllPorts() will export all bounds ports from the keeper.
	// If a Genesis here (controller) is exported and then a chain attempts to restart with genesis from both controller + host, it will panic with
	// port already bound for the default types.PortID
	// Can we avoid this by using a controller port key and host port key explicitly in the store?
	suite.Require().Equal([]string{types.PortID, TestPortID}, genesisState.GetPorts())
}
