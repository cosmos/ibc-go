package keeper_test

import (
	"github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/types"
	genesistypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
)

func (suite *KeeperTestSuite) TestInitGenesis() {
	suite.SetupTest()

	interchainAccAddr := icatypes.GenerateAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, TestPortID)
	genesisState := genesistypes.ControllerGenesisState{
		ActiveChannels: []genesistypes.ActiveChannel{
			{
				ConnectionId:        ibctesting.FirstConnectionID,
				PortId:              TestPortID,
				ChannelId:           ibctesting.FirstChannelID,
				IsMiddlewareEnabled: true,
			},
			{
				ConnectionId:        "connection-1",
				PortId:              "test-port-1",
				ChannelId:           "channel-1",
				IsMiddlewareEnabled: false,
			},
		},
		InterchainAccounts: []genesistypes.RegisteredInterchainAccount{
			{
				ConnectionId:   ibctesting.FirstConnectionID,
				PortId:         TestPortID,
				AccountAddress: interchainAccAddr.String(),
			},
		},
		Ports: []string{TestPortID},
	}

	keeper.InitGenesis(suite.chainA.GetContext(), suite.chainA.GetSimApp().ICAControllerKeeper, genesisState)

	channelID, found := suite.chainA.GetSimApp().ICAControllerKeeper.GetActiveChannelID(suite.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
	suite.Require().True(found)
	suite.Require().Equal(ibctesting.FirstChannelID, channelID)

	isMiddlewareEnabled := suite.chainA.GetSimApp().ICAControllerKeeper.IsMiddlewareEnabled(suite.chainA.GetContext(), TestPortID, ibctesting.FirstConnectionID)
	suite.Require().True(isMiddlewareEnabled)

	isMiddlewareDisabled := suite.chainA.GetSimApp().ICAControllerKeeper.IsMiddlewareDisabled(suite.chainA.GetContext(), "test-port-1", "connection-1")
	suite.Require().True(isMiddlewareDisabled)

	accountAdrr, found := suite.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(suite.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
	suite.Require().True(found)
	suite.Require().Equal(interchainAccAddr.String(), accountAdrr)

	expParams := types.NewParams(false)
	params := suite.chainA.GetSimApp().ICAControllerKeeper.GetParams(suite.chainA.GetContext())
	suite.Require().Equal(expParams, params)
}

func (suite *KeeperTestSuite) TestExportGenesis() {
	suite.SetupTest()

	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	suite.Require().NoError(err)

	interchainAccAddr, exists := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
	suite.Require().True(exists)

	genesisState := keeper.ExportGenesis(suite.chainA.GetContext(), suite.chainA.GetSimApp().ICAControllerKeeper)

	suite.Require().Equal(path.EndpointA.ChannelID, genesisState.ActiveChannels[0].ChannelId)
	suite.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.ActiveChannels[0].PortId)
	suite.Require().True(genesisState.ActiveChannels[0].IsMiddlewareEnabled)

	suite.Require().Equal(interchainAccAddr, genesisState.InterchainAccounts[0].AccountAddress)
	suite.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.InterchainAccounts[0].PortId)

	suite.Require().Equal([]string{TestPortID}, genesisState.GetPorts())

	expParams := types.DefaultParams()
	suite.Require().Equal(expParams, genesisState.GetParams())
}
