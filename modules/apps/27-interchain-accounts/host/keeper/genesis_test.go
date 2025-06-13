package keeper_test

import (
	genesistypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/genesis/types"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (suite *KeeperTestSuite) TestInitGenesis() {
	interchainAccAddr := icatypes.GenerateAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, TestPortID)
	genesisState := genesistypes.HostGenesisState{
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
				AccountAddress: interchainAccAddr.String(),
			},
		},
		Port: icatypes.HostPortID,
	}

	keeper.InitGenesis(suite.chainA.GetContext(), *suite.chainA.GetSimApp().ICAHostKeeper, genesisState)

	channelID, found := suite.chainA.GetSimApp().ICAHostKeeper.GetActiveChannelID(suite.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
	suite.Require().True(found)
	suite.Require().Equal(ibctesting.FirstChannelID, channelID)

	accountAdrr, found := suite.chainA.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
	suite.Require().True(found)
	suite.Require().Equal(interchainAccAddr.String(), accountAdrr)

	expParams := genesisState.GetParams()
	params := suite.chainA.GetSimApp().ICAHostKeeper.GetParams(suite.chainA.GetContext())
	suite.Require().Equal(expParams, params)

	store := suite.chainA.GetContext().KVStore(suite.chainA.GetSimApp().GetKey(types.StoreKey))
	suite.Require().True(store.Has(icatypes.KeyPort(icatypes.HostPortID)))
}

func (suite *KeeperTestSuite) TestGenesisParams() {
	testCases := []struct {
		name        string
		input       types.Params
		expPanicMsg string
	}{
		{"success: set default params", types.DefaultParams(), ""},
		{"success: non-default params", types.NewParams(!types.DefaultHostEnabled, []string{"/cosmos.staking.v1beta1.MsgDelegate"}), ""},
		{"success: set empty byte for allow messages", types.NewParams(true, nil), ""},
		{"failure: set empty string for allow messages", types.NewParams(true, []string{""}), "could not set ica host params at genesis: parameter must not contain empty strings: []"},
		{"failure: set space string for allow messages", types.NewParams(true, []string{" "}), "could not set ica host params at genesis: parameter must not contain empty strings: [ ]"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			interchainAccAddr := icatypes.GenerateAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, TestPortID)
			genesisState := genesistypes.HostGenesisState{
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
						AccountAddress: interchainAccAddr.String(),
					},
				},
				Port:   icatypes.HostPortID,
				Params: tc.input,
			}
			if tc.expPanicMsg == "" {
				keeper.InitGenesis(suite.chainA.GetContext(), *suite.chainA.GetSimApp().ICAHostKeeper, genesisState)

				channelID, found := suite.chainA.GetSimApp().ICAHostKeeper.GetActiveChannelID(suite.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
				suite.Require().True(found)
				suite.Require().Equal(ibctesting.FirstChannelID, channelID)

				accountAdrr, found := suite.chainA.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
				suite.Require().True(found)
				suite.Require().Equal(interchainAccAddr.String(), accountAdrr)

				expParams := tc.input
				params := suite.chainA.GetSimApp().ICAHostKeeper.GetParams(suite.chainA.GetContext())
				suite.Require().Equal(expParams, params)
			} else {
				suite.PanicsWithError(tc.expPanicMsg, func() {
					keeper.InitGenesis(suite.chainA.GetContext(), *suite.chainA.GetSimApp().ICAHostKeeper, genesisState)
				})
			}
		})
	}
}

func (suite *KeeperTestSuite) TestExportGenesis() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		suite.SetupTest()

		path := NewICAPath(suite.chainA, suite.chainB, icatypes.EncodingProtobuf, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, TestOwnerAddress)
		suite.Require().NoError(err)

		interchainAccAddr, exists := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
		suite.Require().True(exists)

		genesisState := keeper.ExportGenesis(suite.chainB.GetContext(), *suite.chainB.GetSimApp().ICAHostKeeper)

		suite.Require().Equal(path.EndpointB.ChannelID, genesisState.ActiveChannels[0].ChannelId)
		suite.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.ActiveChannels[0].PortId)

		suite.Require().Equal(interchainAccAddr, genesisState.InterchainAccounts[0].AccountAddress)
		suite.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.InterchainAccounts[0].PortId)

		suite.Require().Equal(icatypes.HostPortID, genesisState.GetPort())

		expParams := types.DefaultParams()
		suite.Require().Equal(expParams, genesisState.GetParams())
	}
}
