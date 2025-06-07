package keeper_test

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/types/query"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const doesnotexist = "doesnotexist"

func (suite *KeeperTestSuite) TestQueryChannel() {
	var (
		req        *types.QueryChannelRequest
		expChannel types.Channel
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryChannelRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryChannelRequest{
					PortId:    "test-port-id",
					ChannelId: "",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"channel not found",
			func() {
				req = &types.QueryChannelRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrapf(types.ErrChannelNotFound, "port-id: test-port-id, channel-id test-channel-id").Error(),
			),
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupConnections()
				path.SetChannelOrdered()

				// init channel
				err := path.EndpointA.ChanOpenInit()
				suite.Require().NoError(err)

				expChannel = path.EndpointA.GetChannel()

				req = &types.QueryChannelRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.Channel(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(&expChannel, res.Channel)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryChannels() {
	var (
		req         *types.QueryChannelsRequest
		expChannels = []*types.IdentifiedChannel(nil)
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"empty pagination",
			func() {
				req = &types.QueryChannelsRequest{}
			},
			nil,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()
				// channel0 on first connection on chainA
				counterparty0 := types.Counterparty{
					PortId:    path.EndpointB.ChannelConfig.PortID,
					ChannelId: path.EndpointB.ChannelID,
				}

				// path1 creates a second channel on first connection on chainA
				path1 := ibctesting.NewPath(suite.chainA, suite.chainB)
				path1.SetChannelOrdered()
				path1.EndpointA.ClientID = path.EndpointA.ClientID
				path1.EndpointB.ClientID = path.EndpointB.ClientID
				path1.EndpointA.ConnectionID = path.EndpointA.ConnectionID
				path1.EndpointB.ConnectionID = path.EndpointB.ConnectionID

				suite.coordinator.CreateMockChannels(path1)
				counterparty1 := types.Counterparty{
					PortId:    path1.EndpointB.ChannelConfig.PortID,
					ChannelId: path1.EndpointB.ChannelID,
				}

				channel0 := types.NewChannel(
					types.OPEN, types.UNORDERED,
					counterparty0, []string{path.EndpointA.ConnectionID}, path.EndpointA.ChannelConfig.Version,
				)
				channel1 := types.NewChannel(
					types.OPEN, types.ORDERED,
					counterparty1, []string{path.EndpointA.ConnectionID}, path1.EndpointA.ChannelConfig.Version,
				)

				idCh0 := types.NewIdentifiedChannel(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel0)
				idCh1 := types.NewIdentifiedChannel(path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, channel1)

				expChannels = []*types.IdentifiedChannel{&idCh0, &idCh1}

				req = &types.QueryChannelsRequest{
					Pagination: &query.PageRequest{
						Key:        nil,
						Limit:      2,
						CountTotal: true,
					},
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.Channels(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(len(expChannels), int(res.Pagination.Total))
				suite.Require().ElementsMatch(expChannels, res.Channels) // order of channels is not guaranteed, due to lexicographical ordering
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryConnectionChannels() {
	var (
		req         *types.QueryConnectionChannelsRequest
		expChannels = []*types.IdentifiedChannel{}
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid connection ID",
			func() {
				req = &types.QueryConnectionChannelsRequest{
					Connection: "",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()
				// channel0 on first connection on chainA
				counterparty0 := types.Counterparty{
					PortId:    path.EndpointB.ChannelConfig.PortID,
					ChannelId: path.EndpointB.ChannelID,
				}

				// path1 creates a second channel on first connection on chainA
				path1 := ibctesting.NewPath(suite.chainA, suite.chainB)
				path1.SetChannelOrdered()
				path1.EndpointA.ClientID = path.EndpointA.ClientID
				path1.EndpointB.ClientID = path.EndpointB.ClientID
				path1.EndpointA.ConnectionID = path.EndpointA.ConnectionID
				path1.EndpointB.ConnectionID = path.EndpointB.ConnectionID

				suite.coordinator.CreateMockChannels(path1)
				counterparty1 := types.Counterparty{
					PortId:    path1.EndpointB.ChannelConfig.PortID,
					ChannelId: path1.EndpointB.ChannelID,
				}

				channel0 := types.NewChannel(
					types.OPEN, types.UNORDERED,
					counterparty0, []string{path.EndpointA.ConnectionID}, path.EndpointA.ChannelConfig.Version,
				)
				channel1 := types.NewChannel(
					types.OPEN, types.ORDERED,
					counterparty1, []string{path.EndpointA.ConnectionID}, path.EndpointA.ChannelConfig.Version,
				)

				idCh0 := types.NewIdentifiedChannel(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel0)
				idCh1 := types.NewIdentifiedChannel(path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, channel1)

				expChannels = []*types.IdentifiedChannel{&idCh0, &idCh1}

				req = &types.QueryConnectionChannelsRequest{
					Connection: path.EndpointA.ConnectionID,
					Pagination: &query.PageRequest{
						Key:        nil,
						Limit:      2,
						CountTotal: true,
					},
				}
			},
			nil,
		},
		{
			"success, empty response",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()
				expChannels = []*types.IdentifiedChannel(nil)
				req = &types.QueryConnectionChannelsRequest{
					Connection: "externalConnID",
					Pagination: &query.PageRequest{
						Key:        nil,
						Limit:      2,
						CountTotal: false,
					},
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.ConnectionChannels(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expChannels, res.Channels)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryChannelClientState() {
	var (
		req                      *types.QueryChannelClientStateRequest
		expIdentifiedClientState clienttypes.IdentifiedClientState
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryChannelClientStateRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryChannelClientStateRequest{
					PortId:    "test-port-id",
					ChannelId: "",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"channel not found",
			func() {
				req = &types.QueryChannelClientStateRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrapf(types.ErrChannelNotFound, "port-id: test-port-id, channel-id: test-channel-id").Error(),
			),
		},
		{
			"connection not found",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				channel := path.EndpointA.GetChannel()
				// update channel to reference a connection that does not exist
				channel.ConnectionHops[0] = doesnotexist

				// set connection hops to wrong connection ID
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)

				req = &types.QueryChannelClientStateRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			}, status.Error(
				codes.NotFound,
				errorsmod.Wrapf(connectiontypes.ErrConnectionNotFound, "connection-id: doesnotexist").Error(),
			),
		},
		{
			"client state for channel's connection not found",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				// set connection to empty so clientID is empty
				suite.chainA.App.GetIBCKeeper().ConnectionKeeper.SetConnection(suite.chainA.GetContext(), path.EndpointA.ConnectionID, connectiontypes.ConnectionEnd{})

				req = &types.QueryChannelClientStateRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			}, status.Error(
				codes.NotFound,
				errorsmod.Wrapf(clienttypes.ErrClientNotFound, "client-id: ").Error(),
			),
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupConnections()
				path.SetChannelOrdered()

				// init channel
				err := path.EndpointA.ChanOpenInit()
				suite.Require().NoError(err)

				expClientState := suite.chainA.GetClientState(path.EndpointA.ClientID)
				expIdentifiedClientState = clienttypes.NewIdentifiedClientState(path.EndpointA.ClientID, expClientState)

				req = &types.QueryChannelClientStateRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.ChannelClientState(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(&expIdentifiedClientState, res.IdentifiedClientState)

				// ensure UnpackInterfaces is defined
				cachedValue := res.IdentifiedClientState.ClientState.GetCachedValue()
				suite.Require().NotNil(cachedValue)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryChannelConsensusState() {
	var (
		req               *types.QueryChannelConsensusStateRequest
		expConsensusState exported.ConsensusState
		expClientID       string
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryChannelConsensusStateRequest{
					PortId:         "",
					ChannelId:      "test-channel-id",
					RevisionNumber: 0,
					RevisionHeight: 1,
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryChannelConsensusStateRequest{
					PortId:         "test-port-id",
					ChannelId:      "",
					RevisionNumber: 0,
					RevisionHeight: 1,
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"channel not found",
			func() {
				req = &types.QueryChannelConsensusStateRequest{
					PortId:         "test-port-id",
					ChannelId:      "test-channel-id",
					RevisionNumber: 0,
					RevisionHeight: 1,
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrapf(types.ErrChannelNotFound, "port-id: test-port-id, channel-id test-channel-id").Error(),
			),
		},
		{
			"connection not found",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				channel := path.EndpointA.GetChannel()
				// update channel to reference a connection that does not exist
				channel.ConnectionHops[0] = doesnotexist

				// set connection hops to wrong connection ID
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)

				req = &types.QueryChannelConsensusStateRequest{
					PortId:         path.EndpointA.ChannelConfig.PortID,
					ChannelId:      path.EndpointA.ChannelID,
					RevisionNumber: 0,
					RevisionHeight: 1,
				}
			}, status.Error(
				codes.NotFound,
				errorsmod.Wrapf(connectiontypes.ErrConnectionNotFound, "connection-id: doesnotexist").Error(),
			),
		},
		{
			"consensus state for channel's connection not found",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				req = &types.QueryChannelConsensusStateRequest{
					PortId:         path.EndpointA.ChannelConfig.PortID,
					ChannelId:      path.EndpointA.ChannelID,
					RevisionNumber: 0,
					RevisionHeight: uint64(suite.chainA.GetContext().BlockHeight()), // use current height
				}
			}, status.Error(
				codes.NotFound,
				errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "client-id: 07-tendermint-0").Error(),
			),
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupConnections()
				path.SetChannelOrdered()

				// init channel
				err := path.EndpointA.ChanOpenInit()
				suite.Require().NoError(err)

				expConsensusState, _ = suite.chainA.GetConsensusState(path.EndpointA.ClientID, path.EndpointA.GetClientLatestHeight())
				suite.Require().NotNil(expConsensusState)
				expClientID = path.EndpointA.ClientID

				req = &types.QueryChannelConsensusStateRequest{
					PortId:         path.EndpointA.ChannelConfig.PortID,
					ChannelId:      path.EndpointA.ChannelID,
					RevisionNumber: path.EndpointA.GetClientLatestHeight().GetRevisionNumber(),
					RevisionHeight: path.EndpointA.GetClientLatestHeight().GetRevisionHeight(),
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.ChannelConsensusState(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				consensusState, err := clienttypes.UnpackConsensusState(res.ConsensusState)
				suite.Require().NoError(err)
				suite.Require().Equal(expConsensusState, consensusState)
				suite.Require().Equal(expClientID, res.ClientId)

				// ensure UnpackInterfaces is defined
				cachedValue := res.ConsensusState.GetCachedValue()
				suite.Require().NotNil(cachedValue)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryPacketCommitment() {
	var (
		req           *types.QueryPacketCommitmentRequest
		expCommitment []byte
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryPacketCommitmentRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
					Sequence:  0,
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryPacketCommitmentRequest{
					PortId:    "test-port-id",
					ChannelId: "",
					Sequence:  0,
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid sequence",
			func() {
				req = &types.QueryPacketCommitmentRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
					Sequence:  0,
				}
			},
			status.Error(
				codes.InvalidArgument,
				errors.New("packet sequence cannot be 0").Error(),
			),
		},
		{
			"channel not found",
			func() {
				req = &types.QueryPacketCommitmentRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
					Sequence:  1,
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (test-port-id) channel ID (test-channel-id)").Error(),
			),
		},
		{
			"commitment not found",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()
				expCommitment = []byte("hash")
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, expCommitment)
				req = &types.QueryPacketCommitmentRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
					Sequence:  2,
				}
			},
			status.Error(
				codes.NotFound,
				errors.New("packet commitment hash not found").Error(),
			),
		},
		{
			"invalid ID",
			func() {
				req = &types.QueryPacketCommitmentRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()
				expCommitment = []byte("hash")
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, expCommitment)

				req = &types.QueryPacketCommitmentRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
					Sequence:  1,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.PacketCommitment(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expCommitment, res.Commitment)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryPacketCommitments() {
	var (
		req            *types.QueryPacketCommitmentsRequest
		expCommitments = []*types.PacketState{}
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid ID",
			func() {
				req = &types.QueryPacketCommitmentsRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"channel not found",
			func() {
				req = &types.QueryPacketCommitmentsRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (test-port-id) channel ID (test-channel-id)").Error(),
			),
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				expCommitments = make([]*types.PacketState, 9)

				for i := range uint64(9) {
					commitment := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, i, fmt.Appendf(nil, "hash_%d", i))
					suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), commitment.PortId, commitment.ChannelId, commitment.Sequence, commitment.Data)
					expCommitments[i] = &commitment
				}

				req = &types.QueryPacketCommitmentsRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
					Pagination: &query.PageRequest{
						Key:        nil,
						Limit:      11,
						CountTotal: true,
					},
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.PacketCommitments(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expCommitments, res.Commitments)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryPacketReceipt() {
	var (
		req         *types.QueryPacketReceiptRequest
		expReceived bool
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryPacketReceiptRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
					Sequence:  1,
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryPacketReceiptRequest{
					PortId:    "test-port-id",
					ChannelId: "",
					Sequence:  1,
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid sequence",
			func() {
				req = &types.QueryPacketReceiptRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
					Sequence:  0,
				}
			},
			status.Error(
				codes.InvalidArgument,
				errors.New("packet sequence cannot be 0").Error(),
			),
		},
		{
			"channel not found",
			func() {
				req = &types.QueryPacketReceiptRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
					Sequence:  1,
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (test-port-id) channel ID (test-channel-id)").Error(),
			),
		},
		{
			"success: receipt not found",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1)

				req = &types.QueryPacketReceiptRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
					Sequence:  3,
				}
				expReceived = false
			},
			nil,
		},
		{
			"success: receipt found",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1)

				req = &types.QueryPacketReceiptRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
					Sequence:  1,
				}
				expReceived = true
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.PacketReceipt(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expReceived, res.Received)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryPacketAcknowledgement() {
	var (
		req    *types.QueryPacketAcknowledgementRequest
		expAck []byte
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryPacketAcknowledgementRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
					Sequence:  0,
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryPacketAcknowledgementRequest{
					PortId:    "test-port-id",
					ChannelId: "",
					Sequence:  0,
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid sequence",
			func() {
				req = &types.QueryPacketAcknowledgementRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
					Sequence:  0,
				}
			},
			status.Error(
				codes.InvalidArgument,
				errors.New("packet sequence cannot be 0").Error(),
			),
		},
		{
			"ack not found",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()
				expAck = []byte("hash")
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, expAck)

				req = &types.QueryPacketAcknowledgementRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
					Sequence:  2,
				}
			},
			status.Error(
				codes.NotFound,
				errors.New("packet acknowledgement hash not found").Error(),
			),
		},
		{
			"channel not found",
			func() {
				req = &types.QueryPacketAcknowledgementRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
					Sequence:  1,
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (test-port-id) channel ID (test-channel-id)").Error(),
			),
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()
				expAck = []byte("hash")
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, expAck)

				req = &types.QueryPacketAcknowledgementRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
					Sequence:  1,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.PacketAcknowledgement(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expAck, res.Acknowledgement)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryPacketAcknowledgements() {
	var (
		req                 *types.QueryPacketAcknowledgementsRequest
		expAcknowledgements = []*types.PacketState{}
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid ID",
			func() {
				req = &types.QueryPacketAcknowledgementsRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"channel not found",
			func() {
				req = &types.QueryPacketAcknowledgementsRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (test-port-id) channel ID (test-channel-id)").Error(),
			),
		},
		{
			"success, filtered res",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				var commitments []uint64

				for i := range uint64(100) {
					ack := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, i, fmt.Appendf(nil, "hash_%d", i))
					suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(suite.chainA.GetContext(), ack.PortId, ack.ChannelId, ack.Sequence, ack.Data)

					if i < 10 { // populate the store with 100 and query for 10 specific acks
						expAcknowledgements = append(expAcknowledgements, &ack)
						commitments = append(commitments, ack.Sequence)
					}
				}

				req = &types.QueryPacketAcknowledgementsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: commitments,
					Pagination:                nil,
				}
			},
			nil,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				expAcknowledgements = make([]*types.PacketState, 9)

				for i := range uint64(9) {
					ack := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, i, fmt.Appendf(nil, "hash_%d", i))
					suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(suite.chainA.GetContext(), ack.PortId, ack.ChannelId, ack.Sequence, ack.Data)
					expAcknowledgements[i] = &ack
				}

				req = &types.QueryPacketAcknowledgementsRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
					Pagination: &query.PageRequest{
						Key:        nil,
						Limit:      11,
						CountTotal: true,
					},
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.PacketAcknowledgements(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expAcknowledgements, res.Acknowledgements)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryUnreceivedPackets() {
	var (
		req    *types.QueryUnreceivedPacketsRequest
		expSeq = []uint64(nil)
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:    "test-port-id",
					ChannelId: "",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid seq",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{0},
				}
			},
			status.Error(
				codes.InvalidArgument,
				errors.New("packet sequence 0 cannot be 0").Error(),
			),
		},
		{
			"invalid seq, ordered channel",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetChannelOrdered()
				path.Setup()

				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{0},
				}
			},
			status.Error(
				codes.InvalidArgument,
				errors.New("packet sequence 0 cannot be 0").Error(),
			),
		},
		{
			"channel not found",
			func() {
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:    "invalid-port-id", //nolint:goconst
					ChannelId: "invalid-channel-id",
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrapf(types.ErrChannelNotFound, "port-id: invalid-port-id, channel-id invalid-channel-id").Error(),
			),
		},
		{
			"basic success empty packet commitments",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				expSeq = []uint64(nil)
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{},
				}
			},
			nil,
		},
		{
			"basic success unreceived packet commitments",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				// no ack exists

				expSeq = []uint64{1}
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{1},
				}
			},
			nil,
		},
		{
			"basic success unreceived packet commitments, nothing to relay",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1)

				expSeq = []uint64(nil)
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{1},
				}
			},
			nil,
		},
		{
			"success multiple unreceived packet commitments",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()
				expSeq = []uint64(nil) // reset
				packetCommitments := []uint64{}

				// set packet receipt for every other sequence
				for seq := uint64(1); seq < 10; seq++ {
					packetCommitments = append(packetCommitments, seq)

					if seq%2 == 0 {
						suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq)
					} else {
						expSeq = append(expSeq, seq)
					}
				}

				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: packetCommitments,
				}
			},
			nil,
		},
		{
			"basic success empty packet commitments, ordered channel",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetChannelOrdered()
				path.Setup()

				expSeq = []uint64(nil)
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{},
				}
			},
			nil,
		},
		{
			"basic success unreceived packet commitments, ordered channel",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetChannelOrdered()
				path.Setup()

				// Note: NextSequenceRecv is set to 1 on channel creation.
				expSeq = []uint64{1}
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{1},
				}
			},
			nil,
		},
		{
			"basic success multiple unreceived packet commitments, ordered channel",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetChannelOrdered()
				path.Setup()

				// Exercise scenario from issue #1532. NextSequenceRecv is 5, packet commitments provided are 2, 7, 9, 10.
				// Packet sequence 2 is already received so only sequences 7, 9, 10 should be considered unreceived.
				expSeq = []uint64{7, 9, 10}
				packetCommitments := []uint64{2, 7, 9, 10}
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceRecv(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 5)

				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: packetCommitments,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.UnreceivedPackets(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expSeq, res.Sequences)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryUnreceivedAcks() {
	var (
		req    *types.QueryUnreceivedAcksRequest
		expSeq = []uint64{}
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryUnreceivedAcksRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryUnreceivedAcksRequest{
					PortId:    "test-port-id",
					ChannelId: "",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"channel not found",
			func() {
				req = &types.QueryUnreceivedAcksRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (test-port-id) channel ID (test-channel-id)").Error(),
			),
		},
		{
			"invalid seq",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				req = &types.QueryUnreceivedAcksRequest{
					PortId:             path.EndpointA.ChannelConfig.PortID,
					ChannelId:          path.EndpointA.ChannelID,
					PacketAckSequences: []uint64{0},
				}
			},
			status.Error(
				codes.InvalidArgument,
				errors.New("packet sequence 0 cannot be 0").Error(),
			),
		},
		{
			"basic success unreceived packet acks",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, []byte("commitment"))

				expSeq = []uint64{1}
				req = &types.QueryUnreceivedAcksRequest{
					PortId:             path.EndpointA.ChannelConfig.PortID,
					ChannelId:          path.EndpointA.ChannelID,
					PacketAckSequences: []uint64{1},
				}
			},
			nil,
		},
		{
			"basic success unreceived packet acknowledgements, nothing to relay",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				expSeq = []uint64(nil)
				req = &types.QueryUnreceivedAcksRequest{
					PortId:             path.EndpointA.ChannelConfig.PortID,
					ChannelId:          path.EndpointA.ChannelID,
					PacketAckSequences: []uint64{1},
				}
			},
			nil,
		},
		{
			"success multiple unreceived packet acknowledgements",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()
				expSeq = []uint64{} // reset
				packetAcks := []uint64{}

				// set packet commitment for every other sequence
				for seq := uint64(1); seq < 10; seq++ {
					packetAcks = append(packetAcks, seq)

					if seq%2 == 0 {
						suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq, []byte("commitement"))
						expSeq = append(expSeq, seq)
					}
				}

				req = &types.QueryUnreceivedAcksRequest{
					PortId:             path.EndpointA.ChannelConfig.PortID,
					ChannelId:          path.EndpointA.ChannelID,
					PacketAckSequences: packetAcks,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.UnreceivedAcks(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expSeq, res.Sequences)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryNextSequenceReceive() {
	var (
		req    *types.QueryNextSequenceReceiveRequest
		expSeq uint64
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryNextSequenceReceiveRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryNextSequenceReceiveRequest{
					PortId:    "test-port-id",
					ChannelId: "",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"channel not found",
			func() {
				req = &types.QueryNextSequenceReceiveRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrapf(types.ErrChannelNotFound, "port-id: test-port-id, channel-id test-channel-id").Error(),
			),
		},
		{
			"basic success on unordered channel returns zero",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				expSeq = 0
				req = &types.QueryNextSequenceReceiveRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			nil,
		},
		{
			"basic success on ordered channel returns the set receive sequence",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetChannelOrdered()
				path.Setup()

				expSeq = 3
				seq := uint64(3)
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceRecv(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq)

				req = &types.QueryNextSequenceReceiveRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.NextSequenceReceive(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expSeq, res.NextSequenceReceive)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryNextSequenceSend() {
	var (
		req    *types.QueryNextSequenceSendRequest
		expSeq uint64
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryNextSequenceSendRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryNextSequenceSendRequest{
					PortId:    "test-port-id",
					ChannelId: "",
				}
			},
			status.Error(
				codes.InvalidArgument,
				errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank").Error(),
			),
		},
		{
			"channel not found",
			func() {
				req = &types.QueryNextSequenceSendRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrapf(types.ErrSequenceSendNotFound, "port-id: test-port-id, channel-id test-channel-id").Error(),
			),
		},
		{
			"basic success on unordered channel returns the set send sequence",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				expSeq = 42
				seq := uint64(42)
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceSend(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq)
				req = &types.QueryNextSequenceSendRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			nil,
		},
		{
			"basic success on ordered channel returns the set send sequence",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetChannelOrdered()
				path.Setup()

				expSeq = 3
				seq := uint64(3)
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceSend(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq)

				req = &types.QueryNextSequenceSendRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeper)
			res, err := queryServer.NextSequenceSend(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expSeq, res.NextSequenceSend)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
