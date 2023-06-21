package keeper_test

import (
	"fmt"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

type testCase = struct {
	msg      string
	malleate func()
	expPass  bool
}

// TestChanOpenInit tests the OpenInit handshake call for channels. It uses message passing
// to enter into the appropriate state and then calls ChanOpenInit directly. The channel is
// being created on chainA. The port capability must be created on chainA before ChanOpenInit
// can succeed.
func (s *KeeperTestSuite) TestChanOpenInit() {
	var (
		path                 *ibctesting.Path
		features             []string
		portCap              *capabilitytypes.Capability
		expErrorMsgSubstring string
	)

	testCases := []testCase{
		{"success", func() {
			s.coordinator.SetupConnections(path)
			features = []string{"ORDER_ORDERED", "ORDER_UNORDERED"}
			s.chainA.CreatePortCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = s.chainA.GetPortCapability(ibctesting.MockPort)
		}, true},
		{"channel already exists", func() {
			s.coordinator.Setup(path)
		}, false},
		{"connection doesn't exist", func() {
			// any non-empty values
			path.EndpointA.ConnectionID = "connection-0"
			path.EndpointB.ConnectionID = "connection-0"
		}, false},
		{"capability is incorrect", func() {
			s.coordinator.SetupConnections(path)
			features = []string{"ORDER_ORDERED", "ORDER_UNORDERED"}
			portCap = capabilitytypes.NewCapability(3)
		}, false},
		{"connection version not negotiated", func() {
			s.coordinator.SetupConnections(path)

			// modify connA versions
			conn := path.EndpointA.GetConnection()

			version := connectiontypes.NewVersion("2", []string{"ORDER_ORDERED", "ORDER_UNORDERED"})
			conn.Versions = append(conn.Versions, version)

			s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetConnection(
				s.chainA.GetContext(),
				path.EndpointA.ConnectionID, conn,
			)
			features = []string{"ORDER_ORDERED", "ORDER_UNORDERED"}
			s.chainA.CreatePortCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = s.chainA.GetPortCapability(ibctesting.MockPort)
		}, false},
		{"connection does not support ORDERED channels", func() {
			s.coordinator.SetupConnections(path)

			// modify connA versions to only support UNORDERED channels
			conn := path.EndpointA.GetConnection()

			version := connectiontypes.NewVersion("1", []string{"ORDER_UNORDERED"})
			conn.Versions = []*connectiontypes.Version{version}

			s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetConnection(
				s.chainA.GetContext(),
				path.EndpointA.ConnectionID, conn,
			)
			// NOTE: Opening UNORDERED channels is still expected to pass but ORDERED channels should fail
			features = []string{"ORDER_UNORDERED"}
			s.chainA.CreatePortCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = s.chainA.GetPortCapability(ibctesting.MockPort)
		}, true},
		{
			msg:     "unauthorized client",
			expPass: false,
			malleate: func() {
				expErrorMsgSubstring = "status is Unauthorized"
				s.coordinator.SetupConnections(path)

				// remove client from allowed list
				params := s.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(s.chainA.GetContext())
				params.AllowedClients = []string{}
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetParams(s.chainA.GetContext(), params)

				s.chainA.CreatePortCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
				portCap = s.chainA.GetPortCapability(ibctesting.MockPort)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			// run test for all types of ordering
			for _, order := range []types.Order{types.UNORDERED, types.ORDERED} {
				s.SetupTest() // reset
				path = ibctesting.NewPath(s.chainA, s.chainB)
				path.EndpointA.ChannelConfig.Order = order
				path.EndpointB.ChannelConfig.Order = order
				expErrorMsgSubstring = ""

				tc.malleate()

				counterparty := types.NewCounterparty(ibctesting.MockPort, ibctesting.FirstChannelID)

				channelID, capability, err := s.chainA.App.GetIBCKeeper().ChannelKeeper.ChanOpenInit(
					s.chainA.GetContext(), path.EndpointA.ChannelConfig.Order, []string{path.EndpointA.ConnectionID},
					path.EndpointA.ChannelConfig.PortID, portCap, counterparty, path.EndpointA.ChannelConfig.Version,
				)

				// check if order is supported by channel to determine expected behaviour
				orderSupported := false
				for _, f := range features {
					if f == order.String() {
						orderSupported = true
					}
				}

				// Testcase must have expectedPass = true AND channel order supported before
				// asserting the channel handshake initiation succeeded
				if tc.expPass && orderSupported {
					s.Require().NoError(err)
					s.Require().NotNil(capability)
					s.Require().Equal(types.FormatChannelIdentifier(0), channelID)

					chanCap, ok := s.chainA.App.GetScopedIBCKeeper().GetCapability(
						s.chainA.GetContext(),
						host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, channelID),
					)
					s.Require().True(ok, "could not retrieve channel capability after successful ChanOpenInit")
					s.Require().Equal(chanCap.String(), capability.String(), "channel capability is not correct")
				} else {
					s.Require().Error(err)
					s.Require().Contains(err.Error(), expErrorMsgSubstring)
					s.Require().Nil(capability)
					s.Require().Equal("", channelID)
				}
			}
		})
	}
}

// TestChanOpenTry tests the OpenTry handshake call for channels. It uses message passing
// to enter into the appropriate state and then calls ChanOpenTry directly. The channel
// is being created on chainB. The port capability must be created on chainB before
// ChanOpenTry can succeed.
func (s *KeeperTestSuite) TestChanOpenTry() {
	var (
		path       *ibctesting.Path
		portCap    *capabilitytypes.Capability
		heightDiff uint64
	)

	testCases := []testCase{
		{"success", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			s.chainB.CreatePortCapability(s.chainB.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = s.chainB.GetPortCapability(ibctesting.MockPort)
		}, true},
		{"connection doesn't exist", func() {
			path.EndpointA.ConnectionID = ibctesting.FirstConnectionID
			path.EndpointB.ConnectionID = ibctesting.FirstConnectionID

			// pass capability check
			s.chainB.CreatePortCapability(s.chainB.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = s.chainB.GetPortCapability(ibctesting.MockPort)
		}, false},
		{"connection is not OPEN", func() {
			s.coordinator.SetupClients(path)
			// pass capability check
			s.chainB.CreatePortCapability(s.chainB.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = s.chainB.GetPortCapability(ibctesting.MockPort)

			err := path.EndpointB.ConnOpenInit()
			s.Require().NoError(err)
		}, false},
		{"consensus state not found", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			s.chainB.CreatePortCapability(s.chainB.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = s.chainB.GetPortCapability(ibctesting.MockPort)

			heightDiff = 3 // consensus state doesn't exist at this height
		}, false},
		{"channel verification failed", func() {
			// not creating a channel on chainA will result in an invalid proof of existence
			s.coordinator.SetupConnections(path)
			portCap = s.chainB.GetPortCapability(ibctesting.MockPort)
		}, false},
		{"port capability not found", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			portCap = capabilitytypes.NewCapability(3)
		}, false},
		{"connection version not negotiated", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			// modify connB versions
			conn := path.EndpointB.GetConnection()

			version := connectiontypes.NewVersion("2", []string{"ORDER_ORDERED", "ORDER_UNORDERED"})
			conn.Versions = append(conn.Versions, version)

			s.chainB.App.GetIBCKeeper().ConnectionKeeper.SetConnection(
				s.chainB.GetContext(),
				path.EndpointB.ConnectionID, conn,
			)
			s.chainB.CreatePortCapability(s.chainB.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = s.chainB.GetPortCapability(ibctesting.MockPort)
		}, false},
		{"connection does not support ORDERED channels", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			// modify connA versions to only support UNORDERED channels
			conn := path.EndpointA.GetConnection()

			version := connectiontypes.NewVersion("1", []string{"ORDER_UNORDERED"})
			conn.Versions = []*connectiontypes.Version{version}

			s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetConnection(
				s.chainA.GetContext(),
				path.EndpointA.ConnectionID, conn,
			)
			s.chainA.CreatePortCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = s.chainA.GetPortCapability(ibctesting.MockPort)
		}, false},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest()  // reset
			heightDiff = 0 // must be explicitly changed in malleate
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			if path.EndpointB.ClientID != "" {
				// ensure client is up to date
				err := path.EndpointB.UpdateClient()
				s.Require().NoError(err)
			}

			counterparty := types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, ibctesting.FirstChannelID)

			channelKey := host.ChannelKey(counterparty.PortId, counterparty.ChannelId)
			proof, proofHeight := s.chainA.QueryProof(channelKey)

			channelID, capability, err := s.chainB.App.GetIBCKeeper().ChannelKeeper.ChanOpenTry(
				s.chainB.GetContext(), types.ORDERED, []string{path.EndpointB.ConnectionID},
				path.EndpointB.ChannelConfig.PortID, portCap, counterparty, path.EndpointA.ChannelConfig.Version,
				proof, malleateHeight(proofHeight, heightDiff),
			)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(capability)

				chanCap, ok := s.chainB.App.GetScopedIBCKeeper().GetCapability(
					s.chainB.GetContext(),
					host.ChannelCapabilityPath(path.EndpointB.ChannelConfig.PortID, channelID),
				)
				s.Require().True(ok, "could not retrieve channel capapbility after successful ChanOpenTry")
				s.Require().Equal(chanCap.String(), capability.String(), "channel capability is not correct")
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// TestChanOpenAck tests the OpenAck handshake call for channels. It uses message passing
// to enter into the appropriate state and then calls ChanOpenAck directly. The handshake
// call is occurring on chainA.
func (s *KeeperTestSuite) TestChanOpenAck() {
	var (
		path                  *ibctesting.Path
		counterpartyChannelID string
		channelCap            *capabilitytypes.Capability
		heightDiff            uint64
	)

	testCases := []testCase{
		{"success", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"success with empty stored counterparty channel ID", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			// set the channel's counterparty channel identifier to empty string
			channel := path.EndpointA.GetChannel()
			channel.Counterparty.ChannelId = ""

			// use a different channel identifier
			counterpartyChannelID = path.EndpointB.ChannelID

			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"channel doesn't exist", func() {}, false},
		{"channel state is not INIT", func() {
			// create fully open channels on both chains
			s.coordinator.Setup(path)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"connection not found", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			// set the channel's connection hops to wrong connection ID
			channel := path.EndpointA.GetChannel()
			channel.ConnectionHops[0] = doesnotexist
			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
		}, false},
		{"connection is not OPEN", func() {
			s.coordinator.SetupClients(path)

			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// create channel in init
			path.SetChannelOrdered()

			err = path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			s.chainA.CreateChannelCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"consensus state not found", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			heightDiff = 3 // consensus state doesn't exist at this height
		}, false},
		{"invalid counterparty channel identifier", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			counterpartyChannelID = "otheridentifier"

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"channel verification failed", func() {
			// chainB is INIT, chainA in TRYOPEN
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()

			err := path.EndpointB.ChanOpenInit()
			s.Require().NoError(err)

			err = path.EndpointA.ChanOpenTry()
			s.Require().NoError(err)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"channel capability not found", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			channelCap = capabilitytypes.NewCapability(6)
		}, false},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest()              // reset
			counterpartyChannelID = "" // must be explicitly changed in malleate
			heightDiff = 0             // must be explicitly changed
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			if counterpartyChannelID == "" {
				counterpartyChannelID = ibctesting.FirstChannelID
			}

			if path.EndpointA.ClientID != "" {
				// ensure client is up to date
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)
			}

			channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, ibctesting.FirstChannelID)
			proof, proofHeight := s.chainB.QueryProof(channelKey)

			err := s.chainA.App.GetIBCKeeper().ChannelKeeper.ChanOpenAck(
				s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channelCap, path.EndpointB.ChannelConfig.Version, counterpartyChannelID,
				proof, malleateHeight(proofHeight, heightDiff),
			)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// TestChanOpenConfirm tests the OpenAck handshake call for channels. It uses message passing
// to enter into the appropriate state and then calls ChanOpenConfirm directly. The handshake
// call is occurring on chainB.
func (s *KeeperTestSuite) TestChanOpenConfirm() {
	var (
		path       *ibctesting.Path
		channelCap *capabilitytypes.Capability
		heightDiff uint64
	)
	testCases := []testCase{
		{"success", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			s.Require().NoError(err)

			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, true},
		{"channel doesn't exist", func() {}, false},
		{"channel state is not TRYOPEN", func() {
			// create fully open channels on both cahins
			s.coordinator.Setup(path)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"connection not found", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			s.Require().NoError(err)

			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			// set the channel's connection hops to wrong connection ID
			channel := path.EndpointB.GetChannel()
			channel.ConnectionHops[0] = doesnotexist
			s.chainB.App.GetIBCKeeper().ChannelKeeper.SetChannel(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, channel)
		}, false},
		{"connection is not OPEN", func() {
			s.coordinator.SetupClients(path)

			err := path.EndpointB.ConnOpenInit()
			s.Require().NoError(err)

			s.chainB.CreateChannelCapability(s.chainB.GetSimApp().ScopedIBCMockKeeper, path.EndpointB.ChannelConfig.PortID, ibctesting.FirstChannelID)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, ibctesting.FirstChannelID)
		}, false},
		{"consensus state not found", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			s.Require().NoError(err)

			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			heightDiff = 3
		}, false},
		{"channel verification failed", func() {
			// chainA is INIT, chainB in TRYOPEN
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"channel capability not found", func() {
			s.coordinator.SetupConnections(path)
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			s.Require().NoError(err)

			channelCap = capabilitytypes.NewCapability(6)
		}, false},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest()  // reset
			heightDiff = 0 // must be explicitly changed
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			if path.EndpointB.ClientID != "" {
				// ensure client is up to date
				err := path.EndpointB.UpdateClient()
				s.Require().NoError(err)

			}

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, ibctesting.FirstChannelID)
			proof, proofHeight := s.chainA.QueryProof(channelKey)

			err := s.chainB.App.GetIBCKeeper().ChannelKeeper.ChanOpenConfirm(
				s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, ibctesting.FirstChannelID,
				channelCap, proof, malleateHeight(proofHeight, heightDiff),
			)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// TestChanCloseInit tests the initial closing of a handshake on chainA by calling
// ChanCloseInit. Both chains will use message passing to setup OPEN channels.
func (s *KeeperTestSuite) TestChanCloseInit() {
	var (
		path                 *ibctesting.Path
		channelCap           *capabilitytypes.Capability
		expErrorMsgSubstring string
	)

	testCases := []testCase{
		{"success", func() {
			s.coordinator.Setup(path)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"channel doesn't exist", func() {
			// any non-nil values work for connections
			path.EndpointA.ConnectionID = ibctesting.FirstConnectionID
			path.EndpointB.ConnectionID = ibctesting.FirstConnectionID

			path.EndpointA.ChannelID = ibctesting.FirstChannelID
			path.EndpointB.ChannelID = ibctesting.FirstChannelID

			// ensure channel capability check passes
			s.chainA.CreateChannelCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"channel state is CLOSED", func() {
			s.coordinator.Setup(path)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			// close channel
			err := path.EndpointA.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
		}, false},
		{"connection not found", func() {
			s.coordinator.Setup(path)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			// set the channel's connection hops to wrong connection ID
			channel := path.EndpointA.GetChannel()
			channel.ConnectionHops[0] = doesnotexist
			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
		}, false},
		{"connection is not OPEN", func() {
			s.coordinator.SetupClients(path)

			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// create channel in init
			path.SetChannelOrdered()
			err = path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)

			// ensure channel capability check passes
			s.chainA.CreateChannelCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"channel capability not found", func() {
			s.coordinator.Setup(path)
			channelCap = capabilitytypes.NewCapability(3)
		}, false},
		{
			msg:     "unauthorized client",
			expPass: false,
			malleate: func() {
				s.coordinator.Setup(path)
				channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				// remove client from allowed list
				params := s.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(s.chainA.GetContext())
				params.AllowedClients = []string{}
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetParams(s.chainA.GetContext(), params)
				expErrorMsgSubstring = "status is Unauthorized"
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)
			expErrorMsgSubstring = ""

			tc.malleate()

			err := s.chainA.App.GetIBCKeeper().ChannelKeeper.ChanCloseInit(
				s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, ibctesting.FirstChannelID, channelCap,
			)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), expErrorMsgSubstring)
			}
		})
	}
}

// TestChanCloseConfirm tests the confirming closing channel ends by calling ChanCloseConfirm
// on chainB. Both chains will use message passing to setup OPEN channels. ChanCloseInit is
// bypassed on chainA by setting the channel state in the ChannelKeeper.
func (s *KeeperTestSuite) TestChanCloseConfirm() {
	var (
		path       *ibctesting.Path
		channelCap *capabilitytypes.Capability
		heightDiff uint64
	)

	testCases := []testCase{
		{"success", func() {
			s.coordinator.Setup(path)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			err := path.EndpointA.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
		}, true},
		{"channel doesn't exist", func() {
			// any non-nil values work for connections
			path.EndpointA.ChannelID = ibctesting.FirstChannelID
			path.EndpointB.ChannelID = ibctesting.FirstChannelID

			// ensure channel capability check passes
			s.chainB.CreateChannelCapability(s.chainB.GetSimApp().ScopedIBCMockKeeper, path.EndpointB.ChannelConfig.PortID, ibctesting.FirstChannelID)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, ibctesting.FirstChannelID)
		}, false},
		{"channel state is CLOSED", func() {
			s.coordinator.Setup(path)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			err := path.EndpointB.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
		}, false},
		{"connection not found", func() {
			s.coordinator.Setup(path)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			// set the channel's connection hops to wrong connection ID
			channel := path.EndpointB.GetChannel()
			channel.ConnectionHops[0] = doesnotexist
			s.chainB.App.GetIBCKeeper().ChannelKeeper.SetChannel(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, channel)
		}, false},
		{"connection is not OPEN", func() {
			s.coordinator.SetupClients(path)

			err := path.EndpointB.ConnOpenInit()
			s.Require().NoError(err)

			// create channel in init
			path.SetChannelOrdered()
			err = path.EndpointB.ChanOpenInit()
			s.Require().NoError(err)

			// ensure channel capability check passes
			s.chainB.CreateChannelCapability(s.chainB.GetSimApp().ScopedIBCMockKeeper, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"consensus state not found", func() {
			s.coordinator.Setup(path)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			err := path.EndpointA.SetChannelState(types.CLOSED)
			s.Require().NoError(err)

			heightDiff = 3
		}, false},
		{"channel verification failed", func() {
			// channel not closed
			s.coordinator.Setup(path)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"channel capability not found", func() {
			s.coordinator.Setup(path)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			err := path.EndpointA.SetChannelState(types.CLOSED)
			s.Require().NoError(err)

			channelCap = capabilitytypes.NewCapability(3)
		}, false},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest()  // reset
			heightDiff = 0 // must explicitly be changed
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, ibctesting.FirstChannelID)
			proof, proofHeight := s.chainA.QueryProof(channelKey)

			err := s.chainB.App.GetIBCKeeper().ChannelKeeper.ChanCloseConfirm(
				s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, ibctesting.FirstChannelID, channelCap,
				proof, malleateHeight(proofHeight, heightDiff),
			)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func malleateHeight(height exported.Height, diff uint64) exported.Height {
	return clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+diff)
}
