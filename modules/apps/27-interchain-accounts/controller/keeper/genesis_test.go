package keeper_test

import (
	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	genesistypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/genesis/types"
	genesistypesv2 "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/genesis/types/v2"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
)

func (suite *KeeperTestSuite) TestInitGenesis() {
	suite.SetupTest()

	genesisState := genesistypesv2.ControllerGenesisState{
		ActiveChannels: []genesistypes.ActiveChannel{
			{
				ConnectionId: ibctesting.FirstConnectionID,
				PortId:       TestPortID,
				ChannelId:    ibctesting.FirstChannelID,
			},
		},
		InterchainAccounts: []genesistypes.RegisteredInterchainAccount{
			{
				ConnectionId:   ibctesting.FirstConnectionID,
				PortId:         TestPortID,
				AccountAddress: TestAccAddress.String(),
			},
		},
		Ports: []string{TestPortID},
		MiddlewareEnabled: []genesistypesv2.MiddlewareEnabled{
			{
				PortId:    TestPortID,
				ChannelId: ibctesting.FirstChannelID,
			},
		},
	}

	keeper.InitGenesis(suite.chainA.GetContext(), suite.chainA.GetSimApp().ICAControllerKeeper, genesisState)

	channelID, found := suite.chainA.GetSimApp().ICAControllerKeeper.GetActiveChannelID(suite.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
	suite.Require().True(found)
	suite.Require().Equal(ibctesting.FirstChannelID, channelID)

	accountAdrr, found := suite.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(suite.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
	suite.Require().True(found)
	suite.Require().Equal(TestAccAddress.String(), accountAdrr)

	expParams := types.NewParams(false)
	params := suite.chainA.GetSimApp().ICAControllerKeeper.GetParams(suite.chainA.GetContext())
	suite.Require().Equal(expParams, params)

	isMiddlewareEnabled := suite.chainA.GetSimApp().ICAControllerKeeper.IsMiddlewareEnabled(suite.chainA.GetContext(), TestPortID, ibctesting.FirstChannelID)
	suite.Require().True(isMiddlewareEnabled)
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

	suite.Require().Equal([]string{TestPortID}, genesisState.GetPorts())

	expParams := types.DefaultParams()
	suite.Require().Equal(expParams, genesisState.GetParams())

	suite.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.MiddlewareEnabled[0].PortId)
	suite.Require().Equal(path.EndpointA.ChannelID, genesisState.MiddlewareEnabled[0].ChannelId)
}
