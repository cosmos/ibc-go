package keeper_test

import (
	"fmt"

	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"

	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

// TestChannOpenInit tests the OpenInit handshake call for multihop channels.
func (suite *MultihopTestSuite) TestChanOpenInit() {

	var (
		features             []string
		portCap              *capabilitytypes.Capability
		expErrorMsgSubstring string
	)

	testCases := []testCase{
		{"success", func() {
			suite.SetupConnections()
			features = []string{"ORDER_ORDERED", "ORDER_UNORDERED"}
			suite.A().Chain.CreatePortCapability(
				suite.A().Chain.GetSimApp().ScopedIBCMockKeeper,
				suite.A().ChannelConfig.PortID,
			)
			portCap = suite.A().Chain.GetPortCapability(suite.A().ChannelConfig.PortID)
		}, true},
		{"multi-hop channel already exists", func() {
			suite.coord.SetupChannels(suite.chanPath)
		}, false},
		{"connection doesn't exist", func() {
			// any non-empty values
			suite.chanPath.EndpointA.ConnectionID = "connection-0"
			suite.chanPath.EndpointZ.ConnectionID = "connection-0"
		}, false},
		{"capability is incorrect", func() {
			suite.SetupConnections()

			suite.A().Chain.CreatePortCapability(
				suite.A().Chain.GetSimApp().ScopedIBCMockKeeper,
				suite.A().ChannelConfig.PortID,
			)
			portCap = capabilitytypes.NewCapability(42)
		}, false},
		{"connection version not negotiated", func() {
			suite.coord.SetupConnections(suite.chanPath)

			// modify connA versions
			conn := suite.chanPath.EndpointA.GetConnection()

			version := connectiontypes.NewVersion("2", []string{"ORDER_ORDERED", "ORDER_UNORDERED"})
			conn.Versions = append(conn.Versions, version)

			suite.A().Chain.App.GetIBCKeeper().ConnectionKeeper.SetConnection(
				suite.A().Chain.GetContext(),
				suite.chanPath.EndpointA.ConnectionID, conn,
			)
			// features = []string{"ORDER_ORDERED", "ORDER_UNORDERED"}
			suite.A().Chain.CreatePortCapability(suite.A().Chain.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = suite.A().Chain.GetPortCapability(ibctesting.MockPort)
		}, false},
		{"connection does not support ORDERED channels", func() {
			suite.coord.SetupConnections(suite.chanPath)

			// modify connA versions to only support UNORDERED channels
			conn := suite.chanPath.EndpointA.GetConnection()

			version := connectiontypes.NewVersion("1", []string{"ORDER_UNORDERED"})
			conn.Versions = []*connectiontypes.Version{version}

			suite.A().Chain.App.GetIBCKeeper().ConnectionKeeper.SetConnection(
				suite.A().Chain.GetContext(),
				suite.chanPath.EndpointA.ConnectionID, conn,
			)
			// NOTE: Opening UNORDERED channels is still expected to pass but ORDERED channels should fail
			features = []string{"ORDER_UNORDERED"}
			suite.A().Chain.CreatePortCapability(suite.A().Chain.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = suite.A().Chain.GetPortCapability(ibctesting.MockPort)
		}, true},
		{"unauthorized client", func() {
			expErrorMsgSubstring = "status is Unauthorized"
			suite.coord.SetupConnections(suite.chanPath)

			// remove client from allowed list
			params := suite.A().Chain.App.GetIBCKeeper().ClientKeeper.GetParams(suite.A().Chain.GetContext())
			params.AllowedClients = []string{}
			suite.A().Chain.App.GetIBCKeeper().ClientKeeper.SetParams(suite.A().Chain.GetContext(), params)

			suite.A().Chain.CreatePortCapability(suite.A().Chain.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = suite.A().Chain.GetPortCapability(ibctesting.MockPort)
		}, false},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			// run tests for all types of ordering
			for _, order := range []types.Order{types.ORDERED, types.UNORDERED} {
				suite.SetupTest() // reset
				suite.A().ChannelConfig.Order = order
				suite.Z().ChannelConfig.Order = order
				expErrorMsgSubstring = ""

				tc.malleate()

				// counterparty := types.NewCounterparty(suite.A().ChannelConfig.PortID, ibctesting.FirstChannelID)
				counterparty := types.NewCounterparty(suite.Z().ChannelConfig.PortID, "")
				channelID, cap, err := suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenInit(
					suite.A().Chain.GetContext(),
					suite.A().ChannelConfig.Order,
					[]string{suite.A().ConnectionID},
					suite.A().ChannelConfig.PortID,
					portCap,
					counterparty,
					suite.A().ChannelConfig.Version,
				)

				// check if order is supported by channel to determine expected behaviour
				orderSupported := false
				for _, f := range features {
					if f == order.String() {
						orderSupported = true
					}
				}

				if tc.expPass && orderSupported {
					suite.Require().NoError(err, "channel open init failed")
					suite.Require().NotEmpty(channelID, "channel ID is empty")

					chanCap, ok := suite.A().
						Chain.App.GetScopedIBCKeeper().
						GetCapability(suite.A().Chain.GetContext(), host.ChannelCapabilityPath(suite.A().ChannelConfig.PortID, channelID))
					suite.Require().True(ok, "could not retrieve channel capability after successful ChanOpenInit")
					suite.Require().
						Equal(cap.String(), chanCap.String(), "channel capability is not equal to retrieved capability")
				} else {
					suite.Require().Error(err, "channel open init should fail but passed")
					suite.Require().Contains(err.Error(), expErrorMsgSubstring)
					suite.Require().Equal("", channelID, "channel ID is not empty")
					suite.Require().Nil(cap, "channel capability is not nil")
				}
			}
		})
	}
}

// TestChanOpenTryMultihop tests the OpenTry handshake call for channels over multiple connections.
// It uses message passing to enter into the appropriate state and then calls ChanOpenTry directly.
// The channel is being created on chainB. The port capability must be created on chainB before
// ChanOpenTryMultihop can succeed.
func (suite *MultihopTestSuite) TestChanOpenTryMultihop() {
	var (
		portCap *capabilitytypes.Capability
	)

	testCases := []testCase{
		{"success", func() {
			suite.SetupConnections()
			// manually call ChanOpenInit so we can properly set the connectionHops
			suite.Require().NoError(suite.A().ChanOpenInit())

			suite.Z().Chain.CreatePortCapability(
				suite.Z().Chain.GetSimApp().ScopedIBCKeeper,
				suite.Z().ChannelConfig.PortID,
			)
			portCap = suite.Z().Chain.GetPortCapability(suite.Z().ChannelConfig.PortID)
		}, true},
		// {"connection doesn't exist", func() {
		// 	ibctesting.ChanOpenInit(paths[0].EndpointA, connectionHopsAZ)
		// 	paths[1].EndpointB.ConnectionID = "notfound"
		// 	chainZ := paths[len(paths)-1].EndpointB.Chain
		// 	// pass capability check
		// 	chainZ.CreatePortCapability(suite.chainB.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
		// 	portCap = chainZ.GetPortCapability(ibctesting.MockPort)
		// }, true},
		// {"connection is not OPEN", func() {
		// 	ibctesting.ChanOpenInit(paths[0].EndpointA, connectionHopsAZ)
		// 	// pass capability check
		// 	chainZ := paths[len(paths)-1].EndpointB.Chain
		// 	chainZ.CreatePortCapability(suite.chainB.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
		// 	portCap = chainZ.GetPortCapability(ibctesting.MockPort)

		// 	//err := paths[2].EndpointB.ConnOpenInit()
		// 	//suite.Require().NoError(err)
		// }, false},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			tc.malleate()     // call ChanOpenInit and setup port capabilities
			suite.Z().UpdateAllClients()

			proof := suite.A().QueryChannelProof(nil)
			channelID, cap, err := suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenTry(
				suite.Z().Chain.GetContext(), suite.Z().ChannelConfig.Order,
				suite.Z().GetConnectionHops(),
				suite.Z().ChannelConfig.PortID,
				portCap,
				suite.Z().CounterpartyChannel(),
				suite.A().ChannelConfig.Version,
				proof, suite.Z().GetClientState().GetLatestHeight(),
			)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(cap)

				chanCap, ok := suite.Z().Chain.App.GetScopedIBCKeeper().GetCapability(
					suite.Z().Chain.GetContext(),
					host.ChannelCapabilityPath(suite.Z().ChannelConfig.PortID, channelID),
				)
				suite.Require().True(ok, "could not retrieve channel capapbility after successful ChanOpenTry")
				suite.Require().Equal(chanCap.String(), cap.String(), "channel capability is not correct")
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// TestChanOpenAckMultihop tests the OpenAck handshake call for multihop channels.
// It uses message passing to enter into the appropriate state and then calls
// ChanOpenAck directly. The handshake call is occurring on chainA.
func (suite *MultihopTestSuite) TestChanOpenAckMultihop() {
	var (
		channelCap *capabilitytypes.Capability
	)

	testCases := []testCase{
		{"success", func() {
			suite.SetupConnections()
			fmt.Printf("Pre-ChanOpenInit on chain %s at height=%d\n", suite.A().Chain.ChainID, suite.A().Chain.LastHeader.Header.Height)
			suite.Require().NoError(suite.A().ChanOpenInit())
			initHeight := suite.A().Chain.LastHeader.GetHeight()
			fmt.Printf("ChanOpenInit on chain %s at height=%d\n", suite.A().Chain.ChainID, initHeight.GetRevisionHeight())
			suite.A().Chain.NextBlock() // TODO: why does adding an extra block here make the proof fail???
			fmt.Printf("Using height=%d for ChanOpenTry\n", initHeight.GetRevisionHeight())
			suite.Require().NoError(suite.Z().ChanOpenTry(initHeight))
			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			tc.malleate()     // call ChanOpenInit and setup port capabilities
			suite.A().UpdateAllClients()

			proof := suite.Z().QueryChannelProof(nil)

			err := suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenAck(
				suite.A().Chain.GetContext(), suite.A().ChannelConfig.PortID, suite.A().ChannelID,
				channelCap, suite.Z().ChannelConfig.Version, suite.Z().ChannelID,
				proof, suite.A().GetClientState().GetLatestHeight(),
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// TestChanOpenConfirmMultihop tests the OpenAck handshake call for channels. It uses message passing
// to enter into the appropriate state and then calls ChanOpenConfirm directly. The handshake
// call is occurring on chainB.
func (suite *MultihopTestSuite) TestChanOpenConfirmMultihop() {
	var (
		channelCap *capabilitytypes.Capability
	)

	testCases := []testCase{
		{"success", func() {
			suite.SetupConnections()
			suite.Require().NoError(suite.A().ChanOpenInit())
			suite.Require().NoError(suite.Z().ChanOpenTry(suite.A().Chain.LastHeader.GetHeight()))
			suite.Require().NoError(suite.A().ChanOpenAck(suite.Z().Chain.LastHeader.GetHeight()))
			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			tc.malleate()     // call ChanOpenInit and setup port capabilities
			suite.Z().UpdateAllClients()

			proof := suite.A().QueryChannelProof(nil)

			err := suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenConfirm(
				suite.Z().Chain.GetContext(), suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
				channelCap, proof, suite.Z().GetClientState().GetLatestHeight(),
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// TestChanCloseInitMultihop tests the initial closing of a handshake on chainA by calling
// ChanCloseInit.
func (suite *MultihopTestSuite) TestChanCloseInitMultihop() {
	var (
		channelCap *capabilitytypes.Capability
	)

	testCases := []testCase{
		{"success", func() {
			suite.SetupChannels()
			channelCap = suite.A().Chain.GetChannelCapability(
				suite.A().ChannelConfig.PortID,
				suite.A().ChannelID,
			)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()

			err := suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.ChanCloseInit(
				suite.A().Chain.GetContext(), suite.A().ChannelConfig.PortID, suite.A().ChannelID,
				channelCap,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// TestChanCloseConfirmMultihop tests the confirming closing channel ends by calling ChanCloseConfirm on chainZ.
// ChanCloseInit is bypassed on chainA by setting the channel state in the ChannelKeeper.
func (suite *MultihopTestSuite) TestChanCloseConfirmMultihop() {
	var (
		channelCap *capabilitytypes.Capability
	)

	testCases := []testCase{
		{"success", func() {
			suite.SetupChannels()
			suite.Require().NoError(suite.A().SetChannelClosed())
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID,
				suite.Z().ChannelID,
			)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			suite.Z().UpdateAllClients()

			proof := suite.A().QueryChannelProof(nil)

			err := suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.ChanCloseConfirm(
				suite.Z().Chain.GetContext(), suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
				channelCap,
				proof, suite.Z().GetClientState().GetLatestHeight(),
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
