package keeper_test

import (
	"fmt"

	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
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
			suite.A().ConnectionID = "connection-0"
			suite.Z().ConnectionID = "connection-0"
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
			conn := suite.A().GetConnection()

			version := connectiontypes.NewVersion("2", []string{"ORDER_ORDERED", "ORDER_UNORDERED"})
			conn.Versions = append(conn.Versions, version)

			suite.A().Chain.App.GetIBCKeeper().ConnectionKeeper.SetConnection(
				suite.A().Chain.GetContext(),
				suite.A().ConnectionID, conn,
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
				suite.A().ConnectionID, conn,
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
				suite.SetupTest(5) // reset
				suite.A().ChannelConfig.Order = order
				suite.Z().ChannelConfig.Order = order
				expErrorMsgSubstring = ""

				tc.malleate()

				counterparty := types.NewCounterparty(suite.Z().ChannelConfig.PortID, "")
				channelID, capability, err := suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenInit(
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
						Equal(capability.String(), chanCap.String(), "channel capability is not equal to retrieved capability")
				} else {
					suite.Require().Error(err, "channel open init should fail but passed")
					suite.Require().Contains(err.Error(), expErrorMsgSubstring)
					suite.Require().Equal("", channelID, "channel ID is not empty")
					suite.Require().Nil(capability, "channel capability is not nil")
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
		portCap    *capabilitytypes.Capability
		heightDiff uint64
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
		{"connection doesn't exist", func() {
			suite.chanPath.EndpointA.ConnectionID = ibctesting.FirstConnectionID
			suite.chanPath.EndpointZ.ConnectionID = ibctesting.FirstConnectionID

			// pass capability check
			suite.Z().Chain.CreatePortCapability(suite.Z().Chain.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = suite.Z().Chain.GetPortCapability(ibctesting.MockPort)
		}, false},
		{"connection is not OPEN", func() {
			suite.coord.SetupClients(suite.chanPath)
			// pass capability check
			suite.Z().Chain.CreatePortCapability(suite.Z().Chain.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = suite.Z().Chain.GetPortCapability(ibctesting.MockPort)

			suite.Require().NoError(suite.chanPath.EndpointZ.ConnOpenInit())
		}, false},
		{"consensus state not found", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()
			suite.Require().NoError(suite.A().ChanOpenInit())

			suite.Z().Chain.CreatePortCapability(suite.Z().Chain.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = suite.Z().Chain.GetPortCapability(ibctesting.MockPort)

			heightDiff = 3 // consensus state doesn't exist at this height
		}, false},
		{"channel verification failed", func() {
			// not creating a channel on chainA will result in an invalid proof of existence
			suite.SetupConnections()
			portCap = suite.Z().Chain.GetPortCapability(ibctesting.MockPort)
		}, false},
		{"port capability not found", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()
			suite.Require().NoError(suite.A().ChanOpenInit())

			portCap = capabilitytypes.NewCapability(3)
		}, false},
		{"connection version not negotiated", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()
			suite.Require().NoError(suite.A().ChanOpenInit())

			// modify A counterparty's versions
			chain := suite.A().Endpoint.Counterparty
			conn := chain.GetConnection()

			version := connectiontypes.NewVersion("7", []string{"ORDER_ORDERED", "ORDER_UNORDERED"})
			conn.Versions = append(conn.Versions, version)

			chain.Chain.App.GetIBCKeeper().ConnectionKeeper.SetConnection(
				chain.Chain.GetContext(),
				chain.ConnectionID, conn,
			)

			suite.Z().Chain.CreatePortCapability(suite.Z().Chain.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = suite.Z().Chain.GetPortCapability(ibctesting.MockPort)
		}, false},
		{"connection does not support ORDERED channels", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()
			suite.Require().NoError(suite.A().ChanOpenInit())

			// modify connA versions to only support UNORDERED channels
			conn := suite.chanPath.EndpointA.GetConnection()

			version := connectiontypes.NewVersion("1", []string{"ORDER_UNORDERED"})
			conn.Versions = []*connectiontypes.Version{version}

			suite.A().Chain.App.GetIBCKeeper().ConnectionKeeper.SetConnection(
				suite.A().Chain.GetContext(),
				suite.chanPath.EndpointA.ConnectionID, conn,
			)
			suite.A().Chain.CreatePortCapability(suite.A().Chain.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
			portCap = suite.A().Chain.GetPortCapability(ibctesting.MockPort)
		}, false},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest(5) // reset
			heightDiff = 0
			tc.malleate() // call ChanOpenInit and setup port capabilities

			if suite.chanPath.EndpointZ.ClientID != "" {
				// update client on chainB
				err := suite.chanPath.EndpointZ.UpdateClient()
				suite.Require().NoError(err)
			}

			proof, proofHeight, err := suite.A().QueryChannelProof(suite.A().Chain.LastHeader.GetHeight())

			if tc.expPass {
				suite.Require().NoError(err)
			} else if err != nil {
				return
			}

			channelID, capability, err := suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenTry(
				suite.Z().Chain.GetContext(),
				suite.Z().ChannelConfig.Order,
				suite.Z().GetConnectionHops(),
				suite.Z().ChannelConfig.PortID,
				portCap,
				suite.Z().CounterpartyChannel(),
				suite.A().ChannelConfig.Version,
				proof,
				malleateHeight(proofHeight, heightDiff),
			)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(capability)

				chanCap, ok := suite.Z().Chain.App.GetScopedIBCKeeper().GetCapability(
					suite.Z().Chain.GetContext(),
					host.ChannelCapabilityPath(suite.Z().ChannelConfig.PortID, channelID),
				)
				suite.Require().True(ok, "could not retrieve channel capapbility after successful ChanOpenTry")
				suite.Require().Equal(chanCap.String(), capability.String(), "channel capability is not correct")
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
		counterpartyChannelID string
		channelCap            *capabilitytypes.Capability
		heightDiff            uint64
	)

	testCases := []testCase{
		{"success", func() {
			suite.SetupConnections()
			suite.Require().NoError(suite.A().ChanOpenInit())
			suite.Require().NoError(suite.Z().ChanOpenTry(suite.A().Chain.LastHeader.GetHeight()))
			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
		{"success with empty stored counterparty channel ID", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()

			err := suite.A().ChanOpenInit()
			suite.Require().NoError(err)

			proofHeight := suite.A().Chain.LastHeader.GetHeight()
			err = suite.Z().ChanOpenTry(proofHeight)
			suite.Require().NoError(err)

			// set the channel's counterparty channel identifier to empty string
			channel := suite.A().GetChannel()
			channel.Counterparty.ChannelId = ""

			// use a different channel identifier
			counterpartyChannelID = suite.Z().ChannelID

			suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.A().Chain.GetContext(), suite.A().ChannelConfig.PortID, suite.A().ChannelID, channel)

			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
		{"channel doesn't exist", func() {}, false},
		{"channel state is not INIT", func() {
			// create fully open channels on the chains
			suite.coord.SetupChannels(suite.chanPath)
			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"connection not found", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()
			err := suite.A().ChanOpenInit()
			suite.Require().NoError(err)

			proofHeight := suite.A().Chain.LastHeader.GetHeight()
			err = suite.Z().ChanOpenTry(proofHeight)
			suite.Require().NoError(err)

			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)

			// set the channel's connection hops to wrong connection ID
			channel := suite.A().GetChannel()
			channel.ConnectionHops[0] = doesnotexist
			suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.A().Chain.GetContext(), suite.A().ChannelConfig.PortID, suite.A().ChannelID, channel)
		}, false},
		{"connection is not OPEN", func() {
			suite.coord.SetupClients(suite.chanPath)

			err := suite.A().ConnOpenInit()
			suite.Require().NoError(err)

			// create channel in init
			suite.chanPath.SetChannelOrdered()

			err = suite.A().ChanOpenInit()
			suite.Require().NoError(err)

			suite.A().Chain.CreateChannelCapability(suite.A().Chain.GetSimApp().ScopedIBCMockKeeper, suite.A().ChannelConfig.PortID, suite.A().ChannelID)
			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"consensus state not found", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()

			err := suite.A().ChanOpenInit()
			suite.Require().NoError(err)

			proofHeight := suite.A().Chain.LastHeader.GetHeight()
			err = suite.Z().ChanOpenTry(proofHeight)
			suite.Require().NoError(err)

			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)

			heightDiff = 3 // consensus state doesn't exist at this height
		}, false},
		{"invalid counterparty channel identifier", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()

			err := suite.A().ChanOpenInit()
			suite.Require().NoError(err)

			proofHeight := suite.A().Chain.LastHeader.GetHeight()
			err = suite.Z().ChanOpenTry(proofHeight)
			suite.Require().NoError(err)

			counterpartyChannelID = "otheridentifier"

			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"channel verification failed", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()

			err := suite.Z().ChanOpenInit()
			suite.Require().NoError(err)

			proofHeight := suite.Z().Chain.LastHeader.GetHeight()
			err = suite.A().ChanOpenTry(proofHeight)
			suite.Require().NoError(err)

			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"channel capability not found", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()

			err := suite.A().ChanOpenInit()
			suite.Require().NoError(err)

			proofHeight := suite.A().Chain.LastHeader.GetHeight()
			err = suite.Z().ChanOpenTry(proofHeight)
			suite.Require().NoError(err)

			channelCap = capabilitytypes.NewCapability(6)
		}, false},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest(5)         // reset
			counterpartyChannelID = "" // must be explicitly changed in malleate
			heightDiff = 0             // must be explicitly changed

			tc.malleate() // call ChanOpenInit and setup port capabilities

			if counterpartyChannelID == "" {
				counterpartyChannelID = ibctesting.FirstChannelID
			}

			proof, proofHeight, err := suite.Z().QueryChannelProof(suite.Z().Chain.LastHeader.GetHeight())

			if tc.expPass {
				suite.Require().NoError(err)
			} else if err != nil {
				return
			}

			err = suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenAck(
				suite.A().Chain.GetContext(),
				suite.A().ChannelConfig.PortID,
				suite.A().ChannelID,
				channelCap,
				suite.Z().ChannelConfig.Version,
				counterpartyChannelID,
				proof,
				malleateHeight(proofHeight, heightDiff),
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
		heightDiff uint64
	)

	testCases := []testCase{
		{"success", func() {
			suite.SetupConnections()
			suite.Require().NoError(suite.A().ChanOpenInit())
			suite.Require().NoError(suite.Z().ChanOpenTry(suite.A().Chain.LastHeader.GetHeight()))
			suite.Require().NoError(suite.A().ChanOpenAck(suite.Z().Chain.LastHeader.GetHeight()))
			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, true},
		{"channel doesn't exist", func() {}, false},
		{"channel state is not TRYOPEN", func() {
			// create fully open channels on both cahins
			suite.SetupChannels()
			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, false},
		{"connection not found", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()

			err := suite.A().ChanOpenInit()
			suite.Require().NoError(err)

			proofHeight := suite.A().Chain.LastHeader.GetHeight()
			err = suite.Z().ChanOpenTry(proofHeight)
			suite.Require().NoError(err)

			proofHeight = suite.Z().Chain.LastHeader.GetHeight()
			err = suite.A().ChanOpenAck(proofHeight)
			suite.Require().NoError(err)

			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)

			// set the channel's connection hops to wrong connection ID
			channel := suite.Z().GetChannel()
			channel.ConnectionHops[0] = doesnotexist
			suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.Z().Chain.GetContext(), suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, channel)
		}, false},
		{"connection is not OPEN", func() {
			suite.coord.SetupClients(suite.chanPath)

			err := suite.Z().ConnOpenInit()
			suite.Require().NoError(err)

			// create channel in init
			suite.chanPath.SetChannelOrdered()

			err = suite.Z().ChanOpenInit()
			suite.Require().NoError(err)

			suite.Z().Chain.CreateChannelCapability(suite.Z().Chain.GetSimApp().ScopedIBCMockKeeper, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, false},
		{"consensus state not found", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()

			err := suite.A().ChanOpenInit()
			suite.Require().NoError(err)

			proofHeight := suite.A().Chain.LastHeader.GetHeight()
			err = suite.Z().ChanOpenTry(proofHeight)
			suite.Require().NoError(err)

			proofHeight = suite.Z().Chain.LastHeader.GetHeight()
			err = suite.A().ChanOpenAck(proofHeight)
			suite.Require().NoError(err)

			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)

			heightDiff = 3
		}, false},
		{"channel verification failed", func() {
			// chainA is INIT, chainZ in TRYOPEN
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()

			err := suite.A().ChanOpenInit()
			suite.Require().NoError(err)

			proofHeight := suite.A().Chain.LastHeader.GetHeight()
			err = suite.Z().ChanOpenTry(proofHeight)
			suite.Require().NoError(err)

			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, false},
		{"channel capability not found", func() {
			suite.SetupConnections()
			suite.chanPath.SetChannelOrdered()

			err := suite.A().ChanOpenInit()
			suite.Require().NoError(err)

			proofHeight := suite.A().Chain.LastHeader.GetHeight()
			err = suite.Z().ChanOpenTry(proofHeight)
			suite.Require().NoError(err)

			proofHeight = suite.Z().Chain.LastHeader.GetHeight()
			err = suite.A().ChanOpenAck(proofHeight)
			suite.Require().NoError(err)

			channelCap = capabilitytypes.NewCapability(6)
		}, false},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest(5) // reset
			heightDiff = 0     // must be explicitly changed
			tc.malleate()      // call ChanOpenInit and setup port capabilities

			proof, proofHeight, err := suite.A().QueryChannelProof(suite.A().Chain.LastHeader.GetHeight())

			if tc.expPass {
				suite.Require().NoError(err)
			} else if err != nil {
				return
			}

			err = suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.ChanOpenConfirm(
				suite.Z().Chain.GetContext(),
				suite.Z().ChannelConfig.PortID,
				suite.Z().ChannelID,
				channelCap,
				proof,
				malleateHeight(proofHeight, heightDiff),
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
	var channelCap *capabilitytypes.Capability

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
			suite.SetupTest(5) // reset

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
	var channelCap *capabilitytypes.Capability

	testCases := []testCase{
		{"success", func() {
			suite.SetupChannels()
			suite.A().SetChannelClosed()
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID,
				suite.Z().ChannelID,
			)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest(5) // reset

			tc.malleate()

			proof, proofHeight, err := suite.A().QueryChannelProof(suite.A().Chain.LastHeader.GetHeight())

			if tc.expPass {
				suite.Require().NoError(err)
			} else if err != nil {
				return
			}

			err = suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.ChanCloseConfirm(
				suite.Z().Chain.GetContext(),
				suite.Z().ChannelConfig.PortID,
				suite.Z().ChannelID,
				channelCap,
				proof, proofHeight,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// TestChanCloseFrozenMultihop tests closing a channel with a frozen client in the channel path.
func (suite *MultihopTestSuite) TestChanCloseFrozenMultihop() {
	var (
		channelCapA     *capabilitytypes.Capability
		channelCapZ     *capabilitytypes.Capability
		frozenEndpointA *ibctesting.EndpointM
		frozenEndpointZ *ibctesting.EndpointM
		clientIDA       string
		clientIDZ       string
		connectionIDA   string
		connectionIDZ   string
	)

	testCases := []testCase{
		{"success", func() {

			var clientState exported.ClientState

			suite.SetupChannels()

			// freeze client on each side of the "misbehaving chain"
			// it is expected that misbehavior be submitted to each chain
			// connected to the "misbehaving chain"
			paths := suite.A().Paths[0:1]
			_, epA := ibctesting.NewEndpointMFromLinkedPaths(paths)
			frozenEndpointA = &epA
			ep := suite.A().Paths[1].EndpointA
			clientState, connectionIDA, clientIDA = ep.GetClientState(), ep.ConnectionID, ep.GetConnection().ClientId
			cs, ok := clientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			cs.FrozenHeight = clienttypes.NewHeight(0, 1)
			frozenEndpointA.Chain.App.GetIBCKeeper().ClientKeeper.SetClientState(frozenEndpointA.Chain.GetContext(), clientIDA, cs)

			// freeze the other client[]
			paths = suite.Z().Paths[0:1]
			_, epZ := ibctesting.NewEndpointMFromLinkedPaths(paths)
			frozenEndpointZ = &epZ
			ep = suite.Z().Paths[1].EndpointA
			clientState, connectionIDZ, clientIDZ = ep.GetClientState(), ep.ConnectionID, ep.GetConnection().ClientId
			cs, ok = clientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			cs.FrozenHeight = clienttypes.NewHeight(0, 1)
			frozenEndpointZ.Chain.App.GetIBCKeeper().ClientKeeper.SetClientState(frozenEndpointZ.Chain.GetContext(), clientIDZ, cs)

			// commit updates
			suite.coord.CommitBlock(frozenEndpointA.Chain, frozenEndpointZ.Chain)

			channelCapA = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
			channelCapZ = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, true},
		{"success long path", func() {
			suite.SetupTest(7) // rebuild path with 7 chains
			var clientState exported.ClientState

			suite.SetupChannels()

			// freeze client on each side of the "misbehaving chain"
			// it is expected that misbehavior be submitted to each chain
			// connected to the "misbehaving chain"
			paths := suite.A().Paths[0:2]
			_, epA := ibctesting.NewEndpointMFromLinkedPaths(paths)
			frozenEndpointA = &epA
			ep := suite.A().Paths[2].EndpointA
			clientState, connectionIDA, clientIDA = ep.GetClientState(), ep.ConnectionID, ep.GetConnection().ClientId
			cs, ok := clientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			cs.FrozenHeight = clienttypes.NewHeight(0, 1)
			frozenEndpointA.Chain.App.GetIBCKeeper().ClientKeeper.SetClientState(frozenEndpointA.Chain.GetContext(), clientIDA, cs)

			// freeze the other client[]
			paths = suite.Z().Paths[0:2]
			_, epZ := ibctesting.NewEndpointMFromLinkedPaths(paths)
			frozenEndpointZ = &epZ
			ep = suite.Z().Paths[2].EndpointA
			clientState, connectionIDZ, clientIDZ = ep.GetClientState(), ep.ConnectionID, ep.GetConnection().ClientId
			cs, ok = clientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			cs.FrozenHeight = clienttypes.NewHeight(0, 1)
			frozenEndpointZ.Chain.App.GetIBCKeeper().ClientKeeper.SetClientState(frozenEndpointZ.Chain.GetContext(), clientIDZ, cs)

			// commit updates
			suite.coord.CommitBlock(frozenEndpointA.Chain, frozenEndpointZ.Chain)

			channelCapA = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
			channelCapZ = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, true},
		{"channel end frozen", func() {
			var clientState exported.ClientState
			suite.SetupChannels()

			// freeze client on each side of the "misbehaving chain"
			// it is expected that misbehavior be submitted to each chain
			// connected to the "misbehaving chain"
			paths := suite.A().Paths[0:3]
			_, epA := ibctesting.NewEndpointMFromLinkedPaths(paths)
			frozenEndpointA = &epA
			ep := suite.A().Paths[len(suite.A().Paths)-1].EndpointA
			clientState, connectionIDA, clientIDA = ep.GetClientState(), ep.ConnectionID, ep.GetConnection().ClientId
			cs, ok := clientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			cs.FrozenHeight = clienttypes.NewHeight(0, 1)
			frozenEndpointA.Chain.App.GetIBCKeeper().ClientKeeper.SetClientState(frozenEndpointA.Chain.GetContext(), clientIDA, cs)

			// commit updates
			suite.coord.CommitBlock(frozenEndpointA.Chain)

			channelCapA = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
		{"freeze not allowed", func() {
			suite.SetupChannels()
			_, ep := ibctesting.NewEndpointMFromLinkedPaths(suite.A().Paths[0:1])
			frozenEndpointA = &ep
			_, ep = ibctesting.NewEndpointMFromLinkedPaths(suite.Z().Paths[0:1])
			frozenEndpointZ = &ep
			channelCapA = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
			channelCapZ = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, false},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest(5) // reset

			frozenEndpointA = nil
			frozenEndpointZ = nil

			tc.malleate()

			// proof of frozen client for chain A
			proofConnection, proofClientState, proofHeight, err := frozenEndpointA.QueryFrozenClientProof(connectionIDA, clientIDA, frozenEndpointA.Chain.LastHeader.GetHeight())
			suite.Require().NoError(err)

			// close the channel on chain A
			err = suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.ChanCloseFrozen(
				suite.A().Chain.GetContext(),
				suite.A().ChannelConfig.PortID,
				suite.A().ChannelID,
				channelCapA,
				proofConnection,
				proofClientState,
				proofHeight)
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

			if frozenEndpointZ != nil {
				// proof of frozen client for chain Z
				proofConnection, proofClientState, proofHeight, err = frozenEndpointZ.QueryFrozenClientProof(connectionIDZ, clientIDZ, frozenEndpointZ.Chain.LastHeader.GetHeight())
				suite.Require().NoError(err)

				// close the channel on chain Z
				err = suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.ChanCloseFrozen(
					suite.Z().Chain.GetContext(),
					suite.Z().ChannelConfig.PortID,
					suite.Z().ChannelID,
					channelCapZ,
					proofConnection,
					proofClientState,
					proofHeight,
				)
				if tc.expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			}
		})
	}
}
