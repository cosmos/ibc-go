package keeper_test

import (
	genesistypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/genesis/types"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestInitGenesis() {
	interchainAccAddr := icatypes.GenerateAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, TestPortID)
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

	keeper.InitGenesis(s.chainA.GetContext(), *s.chainA.GetSimApp().ICAHostKeeper, genesisState)

	channelID, found := s.chainA.GetSimApp().ICAHostKeeper.GetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
	s.Require().True(found)
	s.Require().Equal(ibctesting.FirstChannelID, channelID)

	accountAdrr, found := s.chainA.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
	s.Require().True(found)
	s.Require().Equal(interchainAccAddr.String(), accountAdrr)

	expParams := genesisState.GetParams()
	params := s.chainA.GetSimApp().ICAHostKeeper.GetParams(s.chainA.GetContext())
	s.Require().Equal(expParams, params)

	store := s.chainA.GetContext().KVStore(s.chainA.GetSimApp().GetKey(types.StoreKey))
	s.Require().True(store.Has(icatypes.KeyPort(icatypes.HostPortID)))
}

func (s *KeeperTestSuite) TestGenesisParams() {
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
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			interchainAccAddr := icatypes.GenerateAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, TestPortID)
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
				keeper.InitGenesis(s.chainA.GetContext(), *s.chainA.GetSimApp().ICAHostKeeper, genesisState)

				channelID, found := s.chainA.GetSimApp().ICAHostKeeper.GetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
				s.Require().True(found)
				s.Require().Equal(ibctesting.FirstChannelID, channelID)

				accountAdrr, found := s.chainA.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, TestPortID)
				s.Require().True(found)
				s.Require().Equal(interchainAccAddr.String(), accountAdrr)

				expParams := tc.input
				params := s.chainA.GetSimApp().ICAHostKeeper.GetParams(s.chainA.GetContext())
				s.Require().Equal(expParams, params)
			} else {
				s.PanicsWithError(tc.expPanicMsg, func() {
					keeper.InitGenesis(s.chainA.GetContext(), *s.chainA.GetSimApp().ICAHostKeeper, genesisState)
				})
			}
		})
	}
}

func (s *KeeperTestSuite) TestExportGenesis() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest()

		path := NewICAPath(s.chainA, s.chainB, icatypes.EncodingProtobuf, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, TestOwnerAddress)
		s.Require().NoError(err)

		interchainAccAddr, exists := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
		s.Require().True(exists)

		genesisState := keeper.ExportGenesis(s.chainB.GetContext(), *s.chainB.GetSimApp().ICAHostKeeper)

		s.Require().Equal(path.EndpointB.ChannelID, genesisState.ActiveChannels[0].ChannelId)
		s.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.ActiveChannels[0].PortId)

		s.Require().Equal(interchainAccAddr, genesisState.InterchainAccounts[0].AccountAddress)
		s.Require().Equal(path.EndpointA.ChannelConfig.PortID, genesisState.InterchainAccounts[0].PortId)

		s.Require().Equal(icatypes.HostPortID, genesisState.GetPort())

		expParams := types.DefaultParams()
		s.Require().Equal(expParams, genesisState.GetParams())
	}
}
