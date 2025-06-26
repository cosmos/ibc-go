package keeper_test

import (
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	genesistypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestInitGenesis() {
	ports := []string{"port1", "port2", "port3"}

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"success", func() {},
		},
	}

	interchainAccAddr := icatypes.GenerateAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, TestPortID)
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
		Ports: ports,
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			tc.malleate()

			keeper.InitGenesis(s.chainA.GetContext(), *s.chainA.GetSimApp().ICAControllerKeeper, genesisState)

			channelID, found := s.chainA.GetSimApp().ICAControllerKeeper.GetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
			s.Require().True(found)
			s.Require().Equal(ibctesting.FirstChannelID, channelID)

			isMiddlewareEnabled := s.chainA.GetSimApp().ICAControllerKeeper.IsMiddlewareEnabled(s.chainA.GetContext(), TestPortID, ibctesting.FirstConnectionID)
			s.Require().True(isMiddlewareEnabled)

			isMiddlewareDisabled := s.chainA.GetSimApp().ICAControllerKeeper.IsMiddlewareDisabled(s.chainA.GetContext(), "test-port-1", "connection-1")
			s.Require().True(isMiddlewareDisabled)

			accountAdrr, found := s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
			s.Require().True(found)
			s.Require().Equal(interchainAccAddr.String(), accountAdrr)

			expParams := types.NewParams(false)
			params := s.chainA.GetSimApp().ICAControllerKeeper.GetParams(s.chainA.GetContext())
			s.Require().Equal(expParams, params)

			for _, port := range ports {
				store := s.chainA.GetContext().KVStore(s.chainA.GetSimApp().GetKey(types.StoreKey))
				s.Require().True(store.Has(icatypes.KeyPort(port)))
			}
		})
	}
}

func (s *KeeperTestSuite) TestExportGenesis() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest()

		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, TestOwnerAddress)
		s.Require().NoError(err)

		interchainAccAddr, exists := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
		s.Require().True(exists)

		genesisState := keeper.ExportGenesis(s.chainA.GetContext(), *s.chainA.GetSimApp().ICAControllerKeeper)

		s.Require().Equal(path.EndpointA.ChannelID, genesisState.ActiveChannels[0].ChannelId)
		s.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.ActiveChannels[0].PortId)
		s.Require().True(genesisState.ActiveChannels[0].IsMiddlewareEnabled)

		s.Require().Equal(interchainAccAddr, genesisState.InterchainAccounts[0].AccountAddress)
		s.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.InterchainAccounts[0].PortId)

		s.Require().Equal([]string{TestPortID}, genesisState.GetPorts())

		expParams := types.DefaultParams()
		s.Require().Equal(expParams, genesisState.GetParams())
	}
}
