package keeper_test

import (
	"fmt"

	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
)

// TestChannOpenInit tests the OpenInit handshake call for multihop channels.
func (suite *MultihopTestSuite) TestChanOpenInit() {

	var portCap *capabilitytypes.Capability

	testCases := []testCase{
		{"success", func() {

			suite.A().Chain.CreatePortCapability(
				suite.A().Chain.GetSimApp().ScopedIBCMockKeeper,
				suite.A().ChannelConfig.PortID,
			)
			portCap = suite.A().Chain.GetPortCapability(suite.A().ChannelConfig.PortID)
		}, true},
		{"capability is incorrect", func() {

			suite.A().Chain.CreatePortCapability(
				suite.A().Chain.GetSimApp().ScopedIBCMockKeeper,
				suite.A().ChannelConfig.PortID,
			)
			portCap = capabilitytypes.NewCapability(42)
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

				if tc.expPass {
					suite.Require().NoError(err, "channel open init failed")
					suite.Require().NotEmpty(channelID, "channel ID is empty")

					chanCap, ok := suite.A().
						Chain.App.GetScopedIBCKeeper().
						GetCapability(suite.A().Chain.GetContext(), host.ChannelCapabilityPath(suite.A().ChannelConfig.PortID, channelID))
					suite.Require().True(ok, "could not retrieve channel capability after successful ChanOpenInit")
					suite.Require().
						Equal(cap.String(), chanCap.String(), "channel capability is not equal to retrieved capability")
					suite.T().Logf("capability: %s\n", cap.String())
				} else {
					suite.Require().Error(err, "channel open init should fail but passed")
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

			_, proof := suite.A().QueryChannelProof()
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

// TestChanOpenTryMultihop tests the OpenTry handshake call for channels over multiple connections.
// It uses message passing to enter into the appropriate state and then calls ChanOpenTry directly.
// The channel is being created on chainB. The port capability must be created on chainB before
// ChanOpenTryMultihop can succeed.
func (suite *MultihopTestSuite) oldTestChanOpenTryMultihop() {
	var (
		paths      ibctesting.LinkedPaths
		portCap    *capabilitytypes.Capability
		heightDiff uint64
		endpointA  *ibctesting.Endpoint
		endpointZ  *ibctesting.Endpoint
	)

	testCases := []testCase{
		{"multihop success", func() {
			// manually call ChanOpenInit so we can properly set the connectionHops
			ibctesting.ChanOpenInit(suite.paths)
			endpointZ.Chain.CreatePortCapability(endpointZ.Chain.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = endpointZ.Chain.GetPortCapability(ibctesting.MockPort)
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
			suite.SetupTest()
			endpointA = suite.paths.A()
			endpointZ = suite.paths.Z()
			heightDiff = 0 // must be explicitly changed in malleate

			tc.malleate() // call ChanOpenInit and setup port capabilities

			counterparty := types.NewCounterparty(endpointA.ChannelConfig.PortID, ibctesting.FirstChannelID)
			channelKey := host.ChannelKey(counterparty.PortId, counterparty.ChannelId)

			// query the channel
			req := &types.QueryChannelRequest{
				PortId:    counterparty.PortId,
				ChannelId: counterparty.ChannelId,
			}

			// receive the channel response and marshal to expected value bytes
			resp, err := endpointA.Chain.App.GetIBCKeeper().Channel(endpointA.Chain.GetContext(), req)
			suite.Require().NoError(err)
			expectedVal, err := resp.Channel.Marshal()
			suite.Require().NoError(err)

			// fmt.Printf("portid=%s channelid=%s\n", counterparty.PortId, counterparty.ChannelId)
			// fmt.Printf("channel: %#v\n", *resp.Channel)
			// fmt.Printf("expectedVal for proof generation: %x\n", expectedVal)

			// generate multihop proof given keypath and value
			proofs, err := ibctesting.GenerateMultiHopProof(suite.paths, channelKey, expectedVal, false)
			suite.Require().NoError(err)

			// verify call to ChanOpenTry completes successfully
			proofHeight := endpointZ.GetClientState().GetLatestHeight()
			proof, err := proofs.Marshal()
			suite.Require().NoError(err)
			channelID, cap, err := endpointZ.Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenTry(
				endpointZ.Chain.GetContext(), endpointA.ChannelConfig.Order, suite.paths.Reverse().GetConnectionHops(),
				endpointZ.ChannelConfig.PortID, portCap, counterparty, endpointA.ChannelConfig.Version,
				proof, malleateHeight(proofHeight, heightDiff),
			)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(cap)

				chanCap, ok := endpointZ.Chain.App.GetScopedIBCKeeper().GetCapability(
					endpointZ.Chain.GetContext(),
					host.ChannelCapabilityPath(endpointZ.ChannelConfig.PortID, channelID),
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
			suite.Require().NoError(suite.A().ChanOpenInit())
			suite.Require().NoError(suite.Z().ChanOpenTry())
			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			tc.malleate()     // call ChanOpenInit and setup port capabilities
			suite.A().UpdateAllClients()

			_, proof := suite.Z().QueryChannelProof()

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
			suite.Require().NoError(suite.A().ChanOpenInit())
			suite.Require().NoError(suite.Z().ChanOpenTry())
			suite.Require().NoError(suite.A().ChanOpenAck())
			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			tc.malleate()     // call ChanOpenInit and setup port capabilities
			suite.Z().UpdateAllClients()

			_, proof := suite.A().QueryChannelProof()

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

// TestChanCloseConfirmMultihop tests the initial closing of a handshake on chainA by calling
// ChanCloseInit. Both chains will use message passing to setup OPEN channels.
func (suite *MultihopTestSuite) TestChanCloseConfirmMultihop() {
	var (
		heightDiff uint64
		channelCap *capabilitytypes.Capability
		endpointA  *ibctesting.Endpoint
		endpointZ  *ibctesting.Endpoint
	)

	testCases := []testCase{
		{"success", func() {
			ibctesting.SetupChannel(suite.paths)

			channelCap = endpointZ.Chain.GetChannelCapability(endpointZ.ChannelConfig.PortID, endpointZ.ChannelID)
			err := endpointA.SetChannelClosed()
			suite.paths.UpdateClients()

			suite.Require().NoError(err)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()
			endpointA = suite.paths.A()
			endpointZ = suite.paths.Z()
			heightDiff = 0

			tc.malleate()

			channelKey := host.ChannelKey(endpointA.ChannelConfig.PortID, ibctesting.FirstChannelID)
			// query the channel
			req := &types.QueryChannelRequest{
				PortId:    endpointA.ChannelConfig.PortID,
				ChannelId: endpointA.ChannelID,
			}

			// receive the channel response and marshal to expected value bytes
			resp, err := endpointA.Chain.App.GetIBCKeeper().Channel(endpointA.Chain.GetContext(), req)
			suite.Require().NoError(err)
			expectedVal, err := resp.Channel.Marshal()
			suite.Require().NoError(err)

			// generate multihop proof given keypath and value
			proofs, err := ibctesting.GenerateMultiHopProof(suite.paths, channelKey, expectedVal, false)
			suite.Require().NoError(err)
			proofHeight := endpointZ.GetClientState().GetLatestHeight()
			proof, err := proofs.Marshal()
			suite.Require().NoError(err)

			err = endpointZ.Chain.App.GetIBCKeeper().ChannelKeeper.ChanCloseConfirm(
				endpointZ.Chain.GetContext(), endpointZ.ChannelConfig.PortID, ibctesting.FirstChannelID, channelCap,
				proof, malleateHeight(proofHeight, heightDiff),
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
