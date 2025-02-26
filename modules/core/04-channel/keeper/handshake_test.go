package keeper_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type testCase = struct {
	msg      string
	malleate func()
	expErr   error
}

// TestChanOpenInit tests the OpenInit handshake call for channels. It uses message passing
// to enter into the appropriate state and then calls ChanOpenInit directly. The channel is
// being created on chainA.
func (suite *KeeperTestSuite) TestChanOpenInit() {
	var (
		path                 *ibctesting.Path
		features             []string
		expErrorMsgSubstring string
	)

	testCases := []testCase{
		{"success", func() {
			path.SetupConnections()
			features = []string{"ORDER_ORDERED", "ORDER_UNORDERED"}
		}, nil},
		{"connection doesn't exist", func() {
			// any non-empty values
			path.EndpointA.ConnectionID = "connection-0"
			path.EndpointB.ConnectionID = "connection-0"
		}, connectiontypes.ErrConnectionNotFound},
		{"connection version not negotiated", func() {
			path.SetupConnections()

			// modify connA versions
			path.EndpointA.UpdateConnection(func(c *connectiontypes.ConnectionEnd) {
				c.Versions = append(c.Versions, connectiontypes.NewVersion("2", []string{"ORDER_ORDERED", "ORDER_UNORDERED"}))
			})

			features = []string{"ORDER_ORDERED", "ORDER_UNORDERED"}
		}, connectiontypes.ErrInvalidVersion},
		{"connection does not support ORDERED channels", func() {
			path.SetupConnections()

			// modify connA versions to only support UNORDERED channels
			path.EndpointA.UpdateConnection(func(c *connectiontypes.ConnectionEnd) {
				c.Versions = []*connectiontypes.Version{connectiontypes.NewVersion("1", []string{"ORDER_UNORDERED"})}
			})

			// NOTE: Opening UNORDERED channels is still expected to pass but ORDERED channels should fail
			features = []string{"ORDER_UNORDERED"}
		}, nil},
		{
			msg:    "unauthorized client",
			expErr: clienttypes.ErrClientNotActive,
			malleate: func() {
				expErrorMsgSubstring = "status is Unauthorized"
				path.SetupConnections()

				// remove client from allowed list
				params := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(suite.chainA.GetContext())
				params.AllowedClients = []string{}
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetParams(suite.chainA.GetContext(), params)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			// run test for all types of ordering
			for _, order := range []types.Order{types.UNORDERED, types.ORDERED} {
				suite.SetupTest() // reset
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.EndpointA.ChannelConfig.Order = order
				path.EndpointB.ChannelConfig.Order = order
				expErrorMsgSubstring = ""

				tc.malleate()

				counterparty := types.NewCounterparty(ibctesting.MockPort, ibctesting.FirstChannelID)

				channelID, err := suite.chainA.App.GetIBCKeeper().ChannelKeeper.ChanOpenInit(
					suite.chainA.GetContext(), path.EndpointA.ChannelConfig.Order, []string{path.EndpointA.ConnectionID},
					path.EndpointA.ChannelConfig.PortID, counterparty, path.EndpointA.ChannelConfig.Version,
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
				if (tc.expErr == nil) && orderSupported {
					suite.Require().NoError(err)
					suite.Require().Equal(types.FormatChannelIdentifier(0), channelID)
				} else {
					suite.Require().Error(err)
					suite.Require().Contains(err.Error(), expErrorMsgSubstring)
					suite.Require().Equal("", channelID)
				}
			}
		})
	}
}

// TestChanOpenTry tests the OpenTry handshake call for channels. It uses message passing
// to enter into the appropriate state and then calls ChanOpenTry directly. The channel
// is being created on chainB.
func (suite *KeeperTestSuite) TestChanOpenTry() {
	var (
		path       *ibctesting.Path
		heightDiff uint64
	)

	testCases := []testCase{
		{"success", func() {
			path.SetupConnections()
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)
		}, nil},
		{"connection doesn't exist", func() {
			path.EndpointA.ConnectionID = ibctesting.FirstConnectionID
			path.EndpointB.ConnectionID = ibctesting.FirstConnectionID
		}, connectiontypes.ErrConnectionNotFound},
		{"connection is not OPEN", func() {
			path.SetupClients()

			err := path.EndpointB.ConnOpenInit()
			suite.Require().NoError(err)
		}, connectiontypes.ErrInvalidConnectionState},
		{"consensus state not found", func() {
			path.SetupConnections()
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)

			heightDiff = 3 // consensus state doesn't exist at this height
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "")},
		{"channel verification failed", func() {
			// not creating a channel on chainA will result in an invalid proof of existence
			path.SetupConnections()
		}, commitmenttypes.ErrInvalidProof},
		{"connection version not negotiated", func() {
			path.SetupConnections()
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)

			// modify connB versions
			path.EndpointB.UpdateConnection(func(c *connectiontypes.ConnectionEnd) {
				c.Versions = append(c.Versions, connectiontypes.NewVersion("2", []string{"ORDER_ORDERED", "ORDER_UNORDERED"}))
			})
		}, connectiontypes.ErrInvalidVersion},
		{"connection does not support ORDERED channels", func() {
			path.SetupConnections()
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)

			// modify connB versions to only support UNORDERED channels
			path.EndpointB.UpdateConnection(func(c *connectiontypes.ConnectionEnd) {
				c.Versions = []*connectiontypes.Version{connectiontypes.NewVersion("1", []string{"ORDER_UNORDERED"})}
			})
		}, connectiontypes.ErrInvalidVersion},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			heightDiff = 0    // must be explicitly changed in malleate
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			if path.EndpointB.ClientID != "" {
				// ensure client is up to date
				err := path.EndpointB.UpdateClient()
				suite.Require().NoError(err)
			}

			counterparty := types.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			channelKey := host.ChannelKey(counterparty.PortId, counterparty.ChannelId)
			proof, proofHeight := suite.chainA.QueryProof(channelKey)

			channelID, err := suite.chainB.App.GetIBCKeeper().ChannelKeeper.ChanOpenTry(
				suite.chainB.GetContext(), types.ORDERED, []string{path.EndpointB.ConnectionID},
				path.EndpointB.ChannelConfig.PortID, counterparty, path.EndpointA.ChannelConfig.Version,
				proof, malleateHeight(proofHeight, heightDiff),
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(channelID)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestChanOpenAck tests the OpenAck handshake call for channels. It uses message passing
// to enter into the appropriate state and then calls ChanOpenAck directly. The handshake
// call is occurring on chainA.
func (suite *KeeperTestSuite) TestChanOpenAck() {
	var (
		path                  *ibctesting.Path
		counterpartyChannelID string
		heightDiff            uint64
	)

	testCases := []testCase{
		{"success", func() {
			path.SetupConnections()
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)
		}, nil},
		{"success with empty stored counterparty channel ID", func() {
			path.SetupConnections()
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			// set the channel's counterparty channel identifier to empty string
			channel := path.EndpointA.GetChannel()
			channel.Counterparty.ChannelId = ""

			// use a different channel identifier
			counterpartyChannelID = path.EndpointB.ChannelID

			suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
		}, nil},
		{"channel doesn't exist", func() {}, errorsmod.Wrap(types.ErrChannelNotFound, "")},
		{"channel state is not INIT", func() {
			// create fully open channels on both chains
			path.Setup()
		}, types.ErrInvalidChannelState},
		{"connection not found", func() {
			path.SetupConnections()
			path.SetChannelOrdered()
			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			// set the channel's connection hops to wrong connection ID
			channel := path.EndpointA.GetChannel()
			channel.ConnectionHops[0] = doesnotexist
			suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
		}, errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, "")},
		{"connection is not OPEN", func() {
			path.SetupClients()

			err := path.EndpointA.ConnOpenInit()
			suite.Require().NoError(err)

			// create channel in init
			path.SetChannelOrdered()

			err = path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)
		}, connectiontypes.ErrInvalidConnectionState},
		{"consensus state not found", func() {
			path.SetupConnections()
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			heightDiff = 3 // consensus state doesn't exist at this height
		}, ibcerrors.ErrInvalidHeight},
		{"invalid counterparty channel identifier", func() {
			path.SetupConnections()
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			counterpartyChannelID = "otheridentifier"
		}, commitmenttypes.ErrInvalidProof},
		{"channel verification failed", func() {
			// chainB is INIT, chainA in TRYOPEN
			path.SetupConnections()
			path.SetChannelOrdered()

			err := path.EndpointB.ChanOpenInit()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanOpenTry()
			suite.Require().NoError(err)
		}, types.ErrInvalidChannelState},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()          // reset
			counterpartyChannelID = "" // must be explicitly changed in malleate
			heightDiff = 0             // must be explicitly changed
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			if counterpartyChannelID == "" {
				counterpartyChannelID = path.EndpointB.ChannelID
			}

			if path.EndpointA.ClientID != "" {
				// ensure client is up to date
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
			}

			channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			proof, proofHeight := suite.chainB.QueryProof(channelKey)

			err := suite.chainA.App.GetIBCKeeper().ChannelKeeper.ChanOpenAck(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.Version, counterpartyChannelID,
				proof, malleateHeight(proofHeight, heightDiff),
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestChanOpenConfirm tests the OpenAck handshake call for channels. It uses message passing
// to enter into the appropriate state and then calls ChanOpenConfirm directly. The handshake
// call is occurring on chainB.
func (suite *KeeperTestSuite) TestChanOpenConfirm() {
	var (
		path       *ibctesting.Path
		heightDiff uint64
	)
	testCases := []testCase{
		{"success", func() {
			path.SetupConnections()
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			suite.Require().NoError(err)
		}, nil},
		{"channel doesn't exist", func() {}, types.ErrChannelNotFound},
		{"channel state is not TRYOPEN", func() {
			// create fully open channels on both chains
			path.Setup()
		}, types.ErrInvalidChannelState},
		{"connection not found", func() {
			path.SetupConnections()
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			suite.Require().NoError(err)

			// set the channel's connection hops to wrong connection ID
			channel := path.EndpointB.GetChannel()
			channel.ConnectionHops[0] = doesnotexist
			suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, channel)
		}, errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, "")},
		{"connection is not OPEN", func() {
			path.SetupClients()

			err := path.EndpointB.ConnOpenInit()
			suite.Require().NoError(err)
		}, types.ErrChannelNotFound},
		{"consensus state not found", func() {
			path.SetupConnections()
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			suite.Require().NoError(err)

			heightDiff = 3
		}, ibcerrors.ErrInvalidHeight},
		{"channel verification failed", func() {
			// chainA is INIT, chainB in TRYOPEN
			path.SetupConnections()
			path.SetChannelOrdered()

			err := path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)
		}, commitmenttypes.ErrInvalidProof},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			heightDiff = 0    // must be explicitly changed
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			if path.EndpointB.ClientID != "" {
				// ensure client is up to date
				err := path.EndpointB.UpdateClient()
				suite.Require().NoError(err)

			}

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proof, proofHeight := suite.chainA.QueryProof(channelKey)

			err := suite.chainB.App.GetIBCKeeper().ChannelKeeper.ChanOpenConfirm(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
				proof, malleateHeight(proofHeight, heightDiff),
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestChanCloseInit tests the initial closing of a handshake on chainA by calling
// ChanCloseInit. Both chains will use message passing to setup OPEN channels.
func (suite *KeeperTestSuite) TestChanCloseInit() {
	var (
		path                 *ibctesting.Path
		expErrorMsgSubstring string
	)

	testCases := []testCase{
		{"success", func() {
			path.Setup()
		}, nil},
		{"channel doesn't exist", func() {
			// any non-nil values work for connections
			path.EndpointA.ConnectionID = ibctesting.FirstConnectionID
			path.EndpointB.ConnectionID = ibctesting.FirstConnectionID

			path.EndpointA.ChannelID = ibctesting.FirstChannelID
			path.EndpointB.ChannelID = ibctesting.FirstChannelID
		}, types.ErrChannelNotFound},
		{"channel state is CLOSED", func() {
			path.Setup()

			// close channel
			path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
		}, types.ErrInvalidChannelState},
		{"connection not found", func() {
			path.Setup()

			// set the channel's connection hops to wrong connection ID
			channel := path.EndpointA.GetChannel()
			channel.ConnectionHops[0] = doesnotexist
			suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
		}, errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, "")},
		{"connection is not OPEN", func() {
			path.SetupClients()

			err := path.EndpointA.ConnOpenInit()
			suite.Require().NoError(err)

			// create channel in init
			path.SetChannelOrdered()
			err = path.EndpointA.ChanOpenInit()
			suite.Require().NoError(err)
		}, connectiontypes.ErrInvalidConnectionState},
		{
			msg:    "unauthorized client",
			expErr: clienttypes.ErrClientNotActive,
			malleate: func() {
				path.Setup()

				// remove client from allowed list
				params := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(suite.chainA.GetContext())
				params.AllowedClients = []string{}
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetParams(suite.chainA.GetContext(), params)
				expErrorMsgSubstring = "status is Unauthorized"
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			expErrorMsgSubstring = ""

			tc.malleate()

			err := suite.chainA.App.GetIBCKeeper().ChannelKeeper.ChanCloseInit(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), expErrorMsgSubstring)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestChanCloseConfirm tests the confirming closing channel ends by calling ChanCloseConfirm
// on chainB. Both chains will use message passing to setup OPEN channels. ChanCloseInit is
// bypassed on chainA by setting the channel state in the ChannelKeeper.
func (suite *KeeperTestSuite) TestChanCloseConfirm() {
	var (
		path       *ibctesting.Path
		heightDiff uint64
	)

	testCases := []testCase{
		{"success", func() {
			path.Setup()
			path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
		}, nil},
		{"channel doesn't exist", func() {
			// any non-nil values work for connections
			path.EndpointA.ChannelID = ibctesting.FirstChannelID
			path.EndpointB.ChannelID = ibctesting.FirstChannelID
		}, errorsmod.Wrap(types.ErrChannelNotFound, "")},
		{"channel state is CLOSED", func() {
			path.Setup()

			path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
		}, types.ErrInvalidChannelState},
		{"connection not found", func() {
			path.Setup()

			// set the channel's connection hops to wrong connection ID
			channel := path.EndpointB.GetChannel()
			channel.ConnectionHops[0] = doesnotexist
			suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, channel)
		}, errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, "")},
		{"connection is not OPEN", func() {
			path.SetupClients()

			err := path.EndpointB.ConnOpenInit()
			suite.Require().NoError(err)

			// create channel in init
			path.SetChannelOrdered()
			err = path.EndpointB.ChanOpenInit()
			suite.Require().NoError(err)
		}, connectiontypes.ErrInvalidConnectionState},
		{"consensus state not found", func() {
			path.Setup()

			path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })

			heightDiff = 3
		}, ibcerrors.ErrInvalidHeight},
		{"channel verification failed", func() {
			// channel not closed
			path.Setup()
		}, ibcerrors.ErrInvalidHeight},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			heightDiff = 0    // must explicitly be changed
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proof, proofHeight := suite.chainA.QueryProof(channelKey)

			ctx := suite.chainB.GetContext()
			err := suite.chainB.App.GetIBCKeeper().ChannelKeeper.ChanCloseConfirm(
				ctx, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
				proof, malleateHeight(proofHeight, heightDiff),
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func malleateHeight(height exported.Height, diff uint64) exported.Height {
	return clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+diff)
}
