package interchain_accounts_test

import (
	ica "github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts"
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
)

func (suite *InterchainAccountsTestSuite) TestInitGenesis() {
	var (
		expectedChannelID string = "channel-0"
	)

	suite.SetupTest()

	genesisState := types.GenesisState{
		Ports: []string{types.PortID, TestPortID},
		ActiveChannels: []*types.ActiveChannel{
			{
				PortId:    TestPortID,
				ChannelId: expectedChannelID,
			},
		},
		InterchainAccounts: []*types.RegisteredInterchainAccount{
			{
				PortId:         TestPortID,
				AccountAddress: TestAccAddress.String(),
			},
		},
	}

	ica.InitGenesis(suite.chainA.GetContext(), suite.chainA.GetSimApp().ICAKeeper, genesisState)

	channelID, found := suite.chainA.GetSimApp().ICAKeeper.GetActiveChannelID(suite.chainA.GetContext(), TestPortID)
	suite.Require().True(found)
	suite.Require().Equal(expectedChannelID, channelID)

	accountAdrr, found := suite.chainA.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainA.GetContext(), TestPortID)
	suite.Require().True(found)
	suite.Require().Equal(TestAccAddress.String(), accountAdrr)
}

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
