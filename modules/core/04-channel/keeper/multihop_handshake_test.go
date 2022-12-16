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

			suite.paths.A().Chain.CreatePortCapability(
				suite.paths.A().Chain.GetSimApp().ScopedIBCMockKeeper,
				suite.paths.A().ChannelConfig.PortID,
			)
			portCap = suite.paths.A().Chain.GetPortCapability(suite.paths.A().ChannelConfig.PortID)
		}, true},
		{"capability is incorrect", func() {

			suite.paths.A().Chain.CreatePortCapability(
				suite.paths.A().Chain.GetSimApp().ScopedIBCMockKeeper,
				suite.paths.A().ChannelConfig.PortID,
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
				suite.paths.A().ChannelConfig.Order = order
				suite.paths.Z().ChannelConfig.Order = order

				tc.malleate()

				// counterparty := types.NewCounterparty(suite.paths.A().ChannelConfig.PortID, ibctesting.FirstChannelID)
				counterparty := types.NewCounterparty(suite.paths.Z().ChannelConfig.PortID, "")
				channelID, cap, err := suite.paths.A().Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenInit(
					suite.paths.A().Chain.GetContext(),
					suite.paths.A().ChannelConfig.Order,
					[]string{suite.paths.A().ConnectionID},
					suite.paths.A().ChannelConfig.PortID,
					portCap,
					counterparty,
					suite.paths.A().ChannelConfig.Version,
				)

				if tc.expPass {
					suite.Require().NoError(err, "channel open init failed")
					suite.Require().NotEmpty(channelID, "channel ID is empty")

					chanCap, ok := suite.paths.A().
						Chain.App.GetScopedIBCKeeper().
						GetCapability(suite.paths.A().Chain.GetContext(), host.ChannelCapabilityPath(suite.paths.A().ChannelConfig.PortID, channelID))
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
		paths      ibctesting.LinkedPaths
		portCap    *capabilitytypes.Capability
		heightDiff uint64
		numChains  int
		endpointA  *ibctesting.Endpoint
		endpointZ  *ibctesting.Endpoint
	)

	testCases := []testCase{
		{"multihop success", func() {
			// manually call ChanOpenInit so we can properly set the connectionHops
			ibctesting.ChanOpenInit(paths)
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

			heightDiff = 0 // must be explicitly changed in malleate
			numChains = 5
			_, paths = ibctesting.CreateLinkedChains(&suite.Suite, numChains)
			endpointA = paths.A()
			endpointZ = paths.Z()

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
			proofs, err := ibctesting.GenerateMultiHopProof(paths, channelKey, expectedVal, false)
			suite.Require().NoError(err)

			// verify call to ChanOpenTry completes successfully
			proofHeight := endpointZ.GetClientState().GetLatestHeight()
			proof, err := proofs.Marshal()
			suite.Require().NoError(err)
			channelID, cap, err := endpointZ.Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenTry(
				endpointZ.Chain.GetContext(), endpointA.ChannelConfig.Order, paths.Reverse().GetConnectionHops(),
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
		paths                 ibctesting.LinkedPaths
		counterpartyChannelID string
		channelCap            *capabilitytypes.Capability
		heightDiff            uint64
		numChains             int
		endpointA             *ibctesting.Endpoint
		endpointZ             *ibctesting.Endpoint
	)

	testCases := []testCase{
		{"success", func() {
			ibctesting.ChanOpenInit(paths)
			ibctesting.ChanOpenTry(paths)
			channelCap = endpointA.Chain.GetChannelCapability(endpointA.ChannelConfig.PortID, endpointA.ChannelID)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			heightDiff = 0 // must be explicitly changed in malleate
			numChains = 5
			_, paths = ibctesting.CreateLinkedChains(&suite.Suite, numChains)
			endpointA = paths.A()
			endpointZ = paths.Z()

			tc.malleate() // call ChanOpenInit and setup port capabilities

			if counterpartyChannelID == "" {
				counterpartyChannelID = ibctesting.FirstChannelID
			}

			channelKey := host.ChannelKey(endpointZ.ChannelConfig.PortID, ibctesting.FirstChannelID)
			// query the channel
			req := &types.QueryChannelRequest{
				PortId:    endpointZ.ChannelConfig.PortID,
				ChannelId: endpointZ.ChannelID,
			}

			// receive the channel response and marshal to expected value bytes
			resp, err := endpointZ.Chain.App.GetIBCKeeper().Channel(endpointZ.Chain.GetContext(), req)
			suite.Require().NoError(err)
			expectedVal, err := resp.Channel.Marshal()
			suite.Require().NoError(err)

			// generate multihop proof given keypath and value
			proofs, err := ibctesting.GenerateMultiHopProof(paths.Reverse(), channelKey, expectedVal, false)
			suite.Require().NoError(err)
			// verify call to ChanOpenTry completes successfully
			proofHeight := endpointA.GetClientState().GetLatestHeight()
			proof, err := proofs.Marshal()
			suite.Require().NoError(err)

			err = endpointA.Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenAck(
				endpointA.Chain.GetContext(), endpointA.ChannelConfig.PortID, endpointA.ChannelID,
				channelCap, endpointZ.ChannelConfig.Version, counterpartyChannelID,
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

// TestChanOpenConfirmMultihop tests the OpenAck handshake call for channels. It uses message passing
// to enter into the appropriate state and then calls ChanOpenConfirm directly. The handshake
// call is occurring on chainB.
func (suite *MultihopTestSuite) TestChanOpenConfirmMultihop() {
	var (
		paths      ibctesting.LinkedPaths
		channelCap *capabilitytypes.Capability
		heightDiff uint64
		numChains  int
		endpointA  *ibctesting.Endpoint
		endpointZ  *ibctesting.Endpoint
	)
	testCases := []testCase{
		{"success", func() {
			ibctesting.ChanOpenInit(paths)
			ibctesting.ChanOpenTry(paths)
			ibctesting.ChanOpenAck(paths)
			channelCap = endpointZ.Chain.GetChannelCapability(endpointZ.ChannelConfig.PortID, endpointZ.ChannelID)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			heightDiff = 0 // must be explicitly changed
			numChains = 5
			_, paths = ibctesting.CreateLinkedChains(&suite.Suite, numChains)
			endpointA = paths.A()
			endpointZ = paths.Z()

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
			proofs, err := ibctesting.GenerateMultiHopProof(paths, channelKey, expectedVal, false)
			suite.Require().NoError(err)
			// verify call to ChanOpenTry completes successfully
			proofHeight := endpointZ.GetClientState().GetLatestHeight()
			proof, err := proofs.Marshal()
			suite.Require().NoError(err)

			err = endpointZ.Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenConfirm(
				endpointZ.Chain.GetContext(), endpointZ.ChannelConfig.PortID, ibctesting.FirstChannelID,
				channelCap, proof, malleateHeight(proofHeight, heightDiff),
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
		paths      ibctesting.LinkedPaths
		heightDiff uint64
		channelCap *capabilitytypes.Capability
		numChains  int
		endpointA  *ibctesting.Endpoint
		endpointZ  *ibctesting.Endpoint
	)

	testCases := []testCase{
		{"success", func() {
			ibctesting.SetupChannel(paths)

			channelCap = endpointZ.Chain.GetChannelCapability(endpointZ.ChannelConfig.PortID, endpointZ.ChannelID)
			err := endpointA.SetChannelClosed()
			paths.UpdateClients()

			suite.Require().NoError(err)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			heightDiff = 0
			numChains = 5
			_, paths = ibctesting.CreateLinkedChains(&suite.Suite, numChains)
			endpointA = paths.A()
			endpointZ = paths.Z()

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
			proofs, err := ibctesting.GenerateMultiHopProof(paths, channelKey, expectedVal, false)
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
