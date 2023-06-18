package keeper_test

import (
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestRegisterInterchainAccount() {
	var (
		owner string
		path  *ibctesting.Path
		err   error
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"port is already bound for owner but capability is claimed by another module",
			func() {
				capability := s.chainA.GetSimApp().IBCKeeper.PortKeeper.BindPort(s.chainA.GetContext(), TestPortID)
				err := s.chainA.GetSimApp().TransferKeeper.ClaimCapability(s.chainA.GetContext(), capability, host.PortPath(TestPortID))
				s.Require().NoError(err)
			},
			false,
		},
		{
			"fails to generate port-id",
			func() {
				owner = ""
			},
			false,
		},
		{
			"MsgChanOpenInit fails - channel is already active & in state OPEN",
			func() {
				portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
				s.Require().NoError(err)

				s.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, portID, path.EndpointA.ChannelID)

				counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				channel := channeltypes.Channel{
					State:          channeltypes.OPEN,
					Ordering:       channeltypes.ORDERED,
					Counterparty:   counterparty,
					ConnectionHops: []string{path.EndpointA.ConnectionID},
					Version:        TestVersion,
				}
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), portID, path.EndpointA.ChannelID, channel)
			},
			false,
		},
	}
	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()

			owner = TestOwnerAddress // must be explicitly changed

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			tc.malleate() // malleate mutates test data

			err = s.chainA.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(s.chainA.GetContext(), path.EndpointA.ConnectionID, owner, TestVersion)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestRegisterSameOwnerMultipleConnections() {
	s.SetupTest()

	owner := TestOwnerAddress

	pathAToB := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(pathAToB)

	pathAToC := NewICAPath(s.chainA, s.chainC)
	s.coordinator.SetupConnections(pathAToC)

	// build ICS27 metadata with connection identifiers for path A->B
	metadata := &icatypes.Metadata{
		Version:                icatypes.Version,
		ControllerConnectionId: pathAToB.EndpointA.ConnectionID,
		HostConnectionId:       pathAToB.EndpointB.ConnectionID,
		Encoding:               icatypes.EncodingProtobuf,
		TxType:                 icatypes.TxTypeSDKMultiMsg,
	}

	err := s.chainA.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(s.chainA.GetContext(), pathAToB.EndpointA.ConnectionID, owner, string(icatypes.ModuleCdc.MustMarshalJSON(metadata)))
	s.Require().NoError(err)

	// build ICS27 metadata with connection identifiers for path A->C
	metadata = &icatypes.Metadata{
		Version:                icatypes.Version,
		ControllerConnectionId: pathAToC.EndpointA.ConnectionID,
		HostConnectionId:       pathAToC.EndpointB.ConnectionID,
		Encoding:               icatypes.EncodingProtobuf,
		TxType:                 icatypes.TxTypeSDKMultiMsg,
	}

	err = s.chainA.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(s.chainA.GetContext(), pathAToC.EndpointA.ConnectionID, owner, string(icatypes.ModuleCdc.MustMarshalJSON(metadata)))
	s.Require().NoError(err)
}
