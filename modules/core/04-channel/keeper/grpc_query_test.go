package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

const doesnotexist = "doesnotexist"

func (s *KeeperTestSuite) TestQueryChannel() {
	var (
		req        *types.QueryChannelRequest
		expChannel types.Channel
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryChannelRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			false,
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryChannelRequest{
					PortId:    "test-port-id",
					ChannelId: "",
				}
			},
			false,
		},
		{
			"channel not found",
			func() {
				req = &types.QueryChannelRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
				}
			},
			false,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupConnections(path)
				path.SetChannelOrdered()

				// init channel
				err := path.EndpointA.ChanOpenInit()
				s.Require().NoError(err)

				expChannel = path.EndpointA.GetChannel()

				req = &types.QueryChannelRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.Channel(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(&expChannel, res.Channel)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryChannels() {
	var (
		req         *types.QueryChannelsRequest
		expChannels = []*types.IdentifiedChannel(nil)
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"empty pagination",
			func() {
				req = &types.QueryChannelsRequest{}
			},
			true,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)
				// channel0 on first connection on chainA
				counterparty0 := types.Counterparty{
					PortId:    path.EndpointB.ChannelConfig.PortID,
					ChannelId: path.EndpointB.ChannelID,
				}

				// path1 creates a second channel on first connection on chainA
				path1 := ibctesting.NewPath(s.chainA, s.chainB)
				path1.SetChannelOrdered()
				path1.EndpointA.ClientID = path.EndpointA.ClientID
				path1.EndpointB.ClientID = path.EndpointB.ClientID
				path1.EndpointA.ConnectionID = path.EndpointA.ConnectionID
				path1.EndpointB.ConnectionID = path.EndpointB.ConnectionID

				s.coordinator.CreateMockChannels(path1)
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
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.Channels(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expChannels, res.Channels)
				s.Require().Equal(len(expChannels), int(res.Pagination.Total))
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryConnectionChannels() {
	var (
		req         *types.QueryConnectionChannelsRequest
		expChannels = []*types.IdentifiedChannel{}
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid connection ID",
			func() {
				req = &types.QueryConnectionChannelsRequest{
					Connection: "",
				}
			},
			false,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)
				// channel0 on first connection on chainA
				counterparty0 := types.Counterparty{
					PortId:    path.EndpointB.ChannelConfig.PortID,
					ChannelId: path.EndpointB.ChannelID,
				}

				// path1 creates a second channel on first connection on chainA
				path1 := ibctesting.NewPath(s.chainA, s.chainB)
				path1.SetChannelOrdered()
				path1.EndpointA.ClientID = path.EndpointA.ClientID
				path1.EndpointB.ClientID = path.EndpointB.ClientID
				path1.EndpointA.ConnectionID = path.EndpointA.ConnectionID
				path1.EndpointB.ConnectionID = path.EndpointB.ConnectionID

				s.coordinator.CreateMockChannels(path1)
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
			true,
		},
		{
			"success, empty response",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)
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
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.ConnectionChannels(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expChannels, res.Channels)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryChannelClientState() {
	var (
		req                      *types.QueryChannelClientStateRequest
		expIdentifiedClientState clienttypes.IdentifiedClientState
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryChannelClientStateRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			false,
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryChannelClientStateRequest{
					PortId:    "test-port-id",
					ChannelId: "",
				}
			},
			false,
		},
		{
			"channel not found",
			func() {
				req = &types.QueryChannelClientStateRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
				}
			},
			false,
		},
		{
			"connection not found",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				channel := path.EndpointA.GetChannel()
				// update channel to reference a connection that does not exist
				channel.ConnectionHops[0] = doesnotexist

				// set connection hops to wrong connection ID
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)

				req = &types.QueryChannelClientStateRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			}, false,
		},
		{
			"client state for channel's connection not found",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				// set connection to empty so clientID is empty
				s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetConnection(s.chainA.GetContext(), path.EndpointA.ConnectionID, connectiontypes.ConnectionEnd{})

				req = &types.QueryChannelClientStateRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			}, false,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupConnections(path)
				path.SetChannelOrdered()

				// init channel
				err := path.EndpointA.ChanOpenInit()
				s.Require().NoError(err)

				expClientState := s.chainA.GetClientState(path.EndpointA.ClientID)
				expIdentifiedClientState = clienttypes.NewIdentifiedClientState(path.EndpointA.ClientID, expClientState)

				req = &types.QueryChannelClientStateRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.ChannelClientState(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(&expIdentifiedClientState, res.IdentifiedClientState)

				// ensure UnpackInterfaces is defined
				cachedValue := res.IdentifiedClientState.ClientState.GetCachedValue()
				s.Require().NotNil(cachedValue)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryChannelConsensusState() {
	var (
		req               *types.QueryChannelConsensusStateRequest
		expConsensusState exported.ConsensusState
		expClientID       string
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
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
			false,
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
			false,
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
			false,
		},
		{
			"connection not found",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				channel := path.EndpointA.GetChannel()
				// update channel to reference a connection that does not exist
				channel.ConnectionHops[0] = doesnotexist

				// set connection hops to wrong connection ID
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)

				req = &types.QueryChannelConsensusStateRequest{
					PortId:         path.EndpointA.ChannelConfig.PortID,
					ChannelId:      path.EndpointA.ChannelID,
					RevisionNumber: 0,
					RevisionHeight: 1,
				}
			}, false,
		},
		{
			"consensus state for channel's connection not found",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				req = &types.QueryChannelConsensusStateRequest{
					PortId:         path.EndpointA.ChannelConfig.PortID,
					ChannelId:      path.EndpointA.ChannelID,
					RevisionNumber: 0,
					RevisionHeight: uint64(s.chainA.GetContext().BlockHeight()), // use current height
				}
			}, false,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupConnections(path)
				path.SetChannelOrdered()

				// init channel
				err := path.EndpointA.ChanOpenInit()
				s.Require().NoError(err)

				clientState := s.chainA.GetClientState(path.EndpointA.ClientID)
				expConsensusState, _ = s.chainA.GetConsensusState(path.EndpointA.ClientID, clientState.GetLatestHeight())
				s.Require().NotNil(expConsensusState)
				expClientID = path.EndpointA.ClientID

				req = &types.QueryChannelConsensusStateRequest{
					PortId:         path.EndpointA.ChannelConfig.PortID,
					ChannelId:      path.EndpointA.ChannelID,
					RevisionNumber: clientState.GetLatestHeight().GetRevisionNumber(),
					RevisionHeight: clientState.GetLatestHeight().GetRevisionHeight(),
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.ChannelConsensusState(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				consensusState, err := clienttypes.UnpackConsensusState(res.ConsensusState)
				s.Require().NoError(err)
				s.Require().Equal(expConsensusState, consensusState)
				s.Require().Equal(expClientID, res.ClientId)

				// ensure UnpackInterfaces is defined
				cachedValue := res.ConsensusState.GetCachedValue()
				s.Require().NotNil(cachedValue)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryPacketCommitment() {
	var (
		req           *types.QueryPacketCommitmentRequest
		expCommitment []byte
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
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
			false,
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
			false,
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
			false,
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
			false,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)
				expCommitment = []byte("hash")
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, expCommitment)

				req = &types.QueryPacketCommitmentRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
					Sequence:  1,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.PacketCommitment(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expCommitment, res.Commitment)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryPacketCommitments() {
	var (
		req            *types.QueryPacketCommitmentsRequest
		expCommitments = []*types.PacketState{}
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid ID",
			func() {
				req = &types.QueryPacketCommitmentsRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			false,
		},
		{
			"success, empty res",
			func() {
				expCommitments = []*types.PacketState(nil)

				req = &types.QueryPacketCommitmentsRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
					Pagination: &query.PageRequest{
						Key:        nil,
						Limit:      2,
						CountTotal: true,
					},
				}
			},
			true,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				expCommitments = make([]*types.PacketState, 9)

				for i := uint64(0); i < 9; i++ {
					commitment := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, i, []byte(fmt.Sprintf("hash_%d", i)))
					s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(s.chainA.GetContext(), commitment.PortId, commitment.ChannelId, commitment.Sequence, commitment.Data)
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
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.PacketCommitments(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expCommitments, res.Commitments)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryPacketReceipt() {
	var (
		req         *types.QueryPacketReceiptRequest
		expReceived bool
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
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
			false,
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
			false,
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
			false,
		},
		{
			"success: receipt not found",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1)

				req = &types.QueryPacketReceiptRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
					Sequence:  3,
				}
				expReceived = false
			},
			true,
		},
		{
			"success: receipt found",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1)

				req = &types.QueryPacketReceiptRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
					Sequence:  1,
				}
				expReceived = true
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.PacketReceipt(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expReceived, res.Received)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryPacketAcknowledgement() {
	var (
		req    *types.QueryPacketAcknowledgementRequest
		expAck []byte
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
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
			false,
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
			false,
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
			false,
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
			false,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)
				expAck = []byte("hash")
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, expAck)

				req = &types.QueryPacketAcknowledgementRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
					Sequence:  1,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.PacketAcknowledgement(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expAck, res.Acknowledgement)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryPacketAcknowledgements() {
	var (
		req                 *types.QueryPacketAcknowledgementsRequest
		expAcknowledgements = []*types.PacketState{}
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid ID",
			func() {
				req = &types.QueryPacketAcknowledgementsRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			false,
		},
		{
			"success, empty res",
			func() {
				expAcknowledgements = []*types.PacketState(nil)

				req = &types.QueryPacketAcknowledgementsRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
					Pagination: &query.PageRequest{
						Key:        nil,
						Limit:      2,
						CountTotal: true,
					},
				}
			},
			true,
		},
		{
			"success, filtered res",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				var commitments []uint64

				for i := uint64(0); i < 100; i++ {
					ack := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, i, []byte(fmt.Sprintf("hash_%d", i)))
					s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(s.chainA.GetContext(), ack.PortId, ack.ChannelId, ack.Sequence, ack.Data)

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
			true,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				expAcknowledgements = make([]*types.PacketState, 9)

				for i := uint64(0); i < 9; i++ {
					ack := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, i, []byte(fmt.Sprintf("hash_%d", i)))
					s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(s.chainA.GetContext(), ack.PortId, ack.ChannelId, ack.Sequence, ack.Data)
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
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.PacketAcknowledgements(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expAcknowledgements, res.Acknowledgements)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryUnreceivedPackets() {
	var (
		req    *types.QueryUnreceivedPacketsRequest
		expSeq = []uint64(nil)
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			false,
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:    "test-port-id",
					ChannelId: "",
				}
			},
			false,
		},
		{
			"invalid seq",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{0},
				}
			},
			false,
		},
		{
			"invalid seq, ordered channel",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetChannelOrdered()
				s.coordinator.Setup(path)

				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{0},
				}
			},
			false,
		},
		{
			"channel not found",
			func() {
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:    "invalid-port-id",
					ChannelId: "invalid-channel-id",
				}
			},
			false,
		},
		{
			"basic success empty packet commitments",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				expSeq = []uint64(nil)
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{},
				}
			},
			true,
		},
		{
			"basic success unreceived packet commitments",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				// no ack exists

				expSeq = []uint64{1}
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{1},
				}
			},
			true,
		},
		{
			"basic success unreceived packet commitments, nothing to relay",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1)

				expSeq = []uint64(nil)
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{1},
				}
			},
			true,
		},
		{
			"success multiple unreceived packet commitments",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)
				expSeq = []uint64(nil) // reset
				packetCommitments := []uint64{}

				// set packet receipt for every other sequence
				for seq := uint64(1); seq < 10; seq++ {
					packetCommitments = append(packetCommitments, seq)

					if seq%2 == 0 {
						s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq)
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
			true,
		},
		{
			"basic success empty packet commitments, ordered channel",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetChannelOrdered()
				s.coordinator.Setup(path)

				expSeq = []uint64(nil)
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{},
				}
			},
			true,
		},
		{
			"basic success unreceived packet commitments, ordered channel",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetChannelOrdered()
				s.coordinator.Setup(path)

				// Note: NextSequenceRecv is set to 1 on channel creation.
				expSeq = []uint64{1}
				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: []uint64{1},
				}
			},
			true,
		},
		{
			"basic success multiple unreceived packet commitments, ordered channel",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetChannelOrdered()
				s.coordinator.Setup(path)

				// Exercise scenario from issue #1532. NextSequenceRecv is 5, packet commitments provided are 2, 7, 9, 10.
				// Packet sequence 2 is already received so only sequences 7, 9, 10 should be considered unreceived.
				expSeq = []uint64{7, 9, 10}
				packetCommitments := []uint64{2, 7, 9, 10}
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceRecv(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 5)

				req = &types.QueryUnreceivedPacketsRequest{
					PortId:                    path.EndpointA.ChannelConfig.PortID,
					ChannelId:                 path.EndpointA.ChannelID,
					PacketCommitmentSequences: packetCommitments,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.UnreceivedPackets(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expSeq, res.Sequences)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryUnreceivedAcks() {
	var (
		req    *types.QueryUnreceivedAcksRequest
		expSeq = []uint64{}
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryUnreceivedAcksRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			false,
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryUnreceivedAcksRequest{
					PortId:    "test-port-id",
					ChannelId: "",
				}
			},
			false,
		},
		{
			"invalid seq",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				req = &types.QueryUnreceivedAcksRequest{
					PortId:             path.EndpointA.ChannelConfig.PortID,
					ChannelId:          path.EndpointA.ChannelID,
					PacketAckSequences: []uint64{0},
				}
			},
			false,
		},
		{
			"basic success unreceived packet acks",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, []byte("commitment"))

				expSeq = []uint64{1}
				req = &types.QueryUnreceivedAcksRequest{
					PortId:             path.EndpointA.ChannelConfig.PortID,
					ChannelId:          path.EndpointA.ChannelID,
					PacketAckSequences: []uint64{1},
				}
			},
			true,
		},
		{
			"basic success unreceived packet acknowledgements, nothing to relay",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				expSeq = []uint64(nil)
				req = &types.QueryUnreceivedAcksRequest{
					PortId:             path.EndpointA.ChannelConfig.PortID,
					ChannelId:          path.EndpointA.ChannelID,
					PacketAckSequences: []uint64{1},
				}
			},
			true,
		},
		{
			"success multiple unreceived packet acknowledgements",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)
				expSeq = []uint64{} // reset
				packetAcks := []uint64{}

				// set packet commitment for every other sequence
				for seq := uint64(1); seq < 10; seq++ {
					packetAcks = append(packetAcks, seq)

					if seq%2 == 0 {
						s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq, []byte("commitement"))
						expSeq = append(expSeq, seq)
					}
				}

				req = &types.QueryUnreceivedAcksRequest{
					PortId:             path.EndpointA.ChannelConfig.PortID,
					ChannelId:          path.EndpointA.ChannelID,
					PacketAckSequences: packetAcks,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.UnreceivedAcks(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expSeq, res.Sequences)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryNextSequenceReceive() {
	var (
		req    *types.QueryNextSequenceReceiveRequest
		expSeq uint64
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryNextSequenceReceiveRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			false,
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryNextSequenceReceiveRequest{
					PortId:    "test-port-id",
					ChannelId: "",
				}
			},
			false,
		},
		{
			"channel not found",
			func() {
				req = &types.QueryNextSequenceReceiveRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
				}
			},
			false,
		},
		{
			"basic success on unordered channel returns zero",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				expSeq = 0
				req = &types.QueryNextSequenceReceiveRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			true,
		},
		{
			"basic success on ordered channel returns the set receive sequence",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetChannelOrdered()
				s.coordinator.Setup(path)

				expSeq = 3
				seq := uint64(3)
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceRecv(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq)

				req = &types.QueryNextSequenceReceiveRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.NextSequenceReceive(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expSeq, res.NextSequenceReceive)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryNextSequenceSend() {
	var (
		req    *types.QueryNextSequenceSendRequest
		expSeq uint64
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid port ID",
			func() {
				req = &types.QueryNextSequenceSendRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			false,
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryNextSequenceSendRequest{
					PortId:    "test-port-id",
					ChannelId: "",
				}
			},
			false,
		},
		{
			"channel not found",
			func() {
				req = &types.QueryNextSequenceSendRequest{
					PortId:    "test-port-id",
					ChannelId: "test-channel-id",
				}
			},
			false,
		},
		{
			"basic success on unordered channel returns zero",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				expSeq = 0
				req = &types.QueryNextSequenceSendRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			true,
		},
		{
			"basic success on ordered channel returns the set send sequence",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetChannelOrdered()
				s.coordinator.Setup(path)

				expSeq = 3
				seq := uint64(3)
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceSend(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq)

				req = &types.QueryNextSequenceSendRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())
			res, err := s.chainA.QueryServer.NextSequenceSend(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expSeq, res.NextSequenceSend)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
