package keeper_test

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

// func (suite *KeeperTestSuite) TestQueryChannel() {
// 	var (
// 		req        *types.QueryChannelRequest
// 		expChannel types.Channel
// 	)

// 	testCases := []struct {
// 		msg      string
// 		malleate func()
// 		expError error
// 	}{
// 		{
// 			"success",
// 			func() {
// 				ctx := suite.chainA.GetContext()
// 				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetChannel(ctx, ibctesting.FirstChannelID, expChannel)

// 				req = &types.QueryChannelRequest{
// 					ChannelId: ibctesting.FirstChannelID,
// 				}
// 			},
// 			nil,
// 		},
// 		{
// 			"req is nil",
// 			func() {
// 				req = nil
// 			},
// 			status.Error(codes.InvalidArgument, "empty request"),
// 		},
// 		{
// 			"invalid channelID",
// 			func() {
// 				req = &types.QueryChannelRequest{}
// 			},
// 			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
// 		},
// 		{
// 			"channel not found",
// 			func() {
// 				req = &types.QueryChannelRequest{
// 					ChannelId: ibctesting.FirstChannelID,
// 				}
// 			},
// 			status.Error(codes.NotFound, fmt.Sprintf("channel-id: %s: channel not found", ibctesting.FirstChannelID)),
// 		},
// 	}

// 	for _, tc := range testCases {
// 		tc := tc

// 		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
// 			suite.SetupTest() // reset

// 			merklePathPrefix := commitmenttypes.NewMerklePath([]byte("prefix"))
// 			expChannel = types.Channel{ClientId: ibctesting.SecondClientID, CounterpartyChannelId: ibctesting.SecondChannelID, MerklePathPrefix: merklePathPrefix}

// 			tc.malleate()

// 			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2)
// 			res, err := queryServer.Channel(suite.chainA.GetContext(), req)

// 			expPass := tc.expError == nil
// 			if expPass {
// 				suite.Require().NoError(err)
// 				suite.Require().NotNil(res)
// 				suite.Require().Equal(expChannel, res.Channel)
// 			} else {
// 				suite.Require().ErrorIs(err, tc.expError)
// 				suite.Require().Nil(res)
// 			}
// 		})
// 	}
// }

// func (suite *KeeperTestSuite) TestQueryChannelClientState() {
// 	var (
// 		req                      *types.QueryChannelClientStateRequest
// 		expIdentifiedClientState clienttypes.IdentifiedClientState
// 	)

// 	testCases := []struct {
// 		msg      string
// 		malleate func()
// 		expError error
// 	}{
// 		{
// 			"success",
// 			func() {
// 				path := ibctesting.NewPath(suite.chainA, suite.chainB)
// 				path.SetupV2()

// 				expClientState := suite.chainA.GetClientState(path.EndpointA.ClientID)
// 				expIdentifiedClientState = clienttypes.NewIdentifiedClientState(path.EndpointA.ClientID, expClientState)

// 				req = &types.QueryChannelClientStateRequest{
// 					ChannelId: path.EndpointA.ChannelID,
// 				}
// 			},
// 			nil,
// 		},
// 		{
// 			"empty request",
// 			func() {
// 				req = nil
// 			},
// 			status.Error(codes.InvalidArgument, "empty request"),
// 		},
// 		{
// 			"invalid channel ID",
// 			func() {
// 				req = &types.QueryChannelClientStateRequest{
// 					ChannelId: "",
// 				}
// 			},
// 			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
// 		},
// 		{
// 			"channel not found",
// 			func() {
// 				req = &types.QueryChannelClientStateRequest{
// 					ChannelId: "test-channel-id",
// 				}
// 			},
// 			status.Error(codes.NotFound, fmt.Sprintf("channel-id: %s: channel not found", "test-channel-id")),
// 		},
// 		{
// 			"client state not found",
// 			func() {
// 				path := ibctesting.NewPath(suite.chainA, suite.chainB)
// 				path.SetupV2()

// 				channel, found := path.EndpointA.Chain.App.GetIBCKeeper().ChannelKeeperV2.GetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelID)
// 				suite.Require().True(found)
// 				channel.ClientId = ibctesting.SecondClientID

// 				path.EndpointA.Chain.App.GetIBCKeeper().ChannelKeeperV2.SetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelID, channel)

// 				req = &types.QueryChannelClientStateRequest{
// 					ChannelId: path.EndpointA.ChannelID,
// 				}
// 			},
// 			status.Error(codes.NotFound, fmt.Sprintf("client-id: %s: light client not found", ibctesting.SecondClientID)),
// 		},
// 	}

// 	for _, tc := range testCases {
// 		tc := tc

// 		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
// 			suite.SetupTest() // reset

// 			tc.malleate()
// 			ctx := suite.chainA.GetContext()

// 			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeperV2)
// 			res, err := queryServer.ChannelClientState(ctx, req)

// 			expPass := tc.expError == nil
// 			if expPass {
// 				suite.Require().NoError(err)
// 				suite.Require().NotNil(res)
// 				suite.Require().Equal(&expIdentifiedClientState, res.IdentifiedClientState)

// 				// ensure UnpackInterfaces is defined
// 				cachedValue := res.IdentifiedClientState.ClientState.GetCachedValue()
// 				suite.Require().NotNil(cachedValue)
// 			} else {
// 				suite.Require().ErrorIs(err, tc.expError)
// 				suite.Require().Nil(res)
// 			}
// 		})
// 	}
// }

// func (suite *KeeperTestSuite) TestQueryChannelConsensusState() {
// 	var (
// 		req               *types.QueryChannelConsensusStateRequest
// 		expConsensusState exported.ConsensusState
// 		expClientID       string
// 	)

// 	testCases := []struct {
// 		msg      string
// 		malleate func()
// 		expError error
// 	}{
// 		{
// 			"success",
// 			func() {
// 				path := ibctesting.NewPath(suite.chainA, suite.chainB)
// 				path.SetupV2()

// 				expConsensusState, _ = suite.chainA.GetConsensusState(path.EndpointA.ClientID, path.EndpointA.GetClientLatestHeight())
// 				suite.Require().NotNil(expConsensusState)
// 				expClientID = path.EndpointA.ClientID

// 				req = &types.QueryChannelConsensusStateRequest{
// 					ChannelId:      path.EndpointA.ChannelID,
// 					RevisionNumber: path.EndpointA.GetClientLatestHeight().GetRevisionNumber(),
// 					RevisionHeight: path.EndpointA.GetClientLatestHeight().GetRevisionHeight(),
// 				}
// 			},
// 			nil,
// 		},
// 		{
// 			"empty request",
// 			func() {
// 				req = nil
// 			},
// 			status.Error(codes.InvalidArgument, "empty request"),
// 		},
// 		{
// 			"invalid channel ID",
// 			func() {
// 				req = &types.QueryChannelConsensusStateRequest{
// 					ChannelId:      "",
// 					RevisionNumber: 0,
// 					RevisionHeight: 1,
// 				}
// 			},
// 			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
// 		},
// 		{
// 			"channel not found",
// 			func() {
// 				req = &types.QueryChannelConsensusStateRequest{
// 					ChannelId:      "test-channel-id",
// 					RevisionNumber: 0,
// 					RevisionHeight: 1,
// 				}
// 			},
// 			status.Error(codes.NotFound, fmt.Sprintf("channel-id: %s: channel not found", "test-channel-id")),
// 		},
// 		{
// 			"consensus state for channel's connection not found",
// 			func() {
// 				path := ibctesting.NewPath(suite.chainA, suite.chainB)
// 				path.SetupV2()

// 				req = &types.QueryChannelConsensusStateRequest{
// 					ChannelId:      path.EndpointA.ChannelID,
// 					RevisionNumber: 0,
// 					RevisionHeight: uint64(suite.chainA.GetContext().BlockHeight()), // use current height
// 				}
// 			},
// 			status.Error(codes.NotFound, fmt.Sprintf("client-id: %s: consensus state not found", "07-tendermint-0")),
// 		},
// 	}

// 	for _, tc := range testCases {
// 		tc := tc

// 		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
// 			suite.SetupTest() // reset

// 			tc.malleate()
// 			ctx := suite.chainA.GetContext()

// 			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeperV2)
// 			res, err := queryServer.ChannelConsensusState(ctx, req)

// 			expPass := tc.expError == nil
// 			if expPass {
// 				suite.Require().NoError(err)
// 				suite.Require().NotNil(res)
// 				consensusState, err := clienttypes.UnpackConsensusState(res.ConsensusState)
// 				suite.Require().NoError(err)
// 				suite.Require().Equal(expConsensusState, consensusState)
// 				suite.Require().Equal(expClientID, res.ClientId)

// 				// ensure UnpackInterfaces is defined
// 				cachedValue := res.ConsensusState.GetCachedValue()
// 				suite.Require().NotNil(cachedValue)
// 			} else {
// 				suite.Require().ErrorIs(err, tc.expError)
// 				suite.Require().Nil(res)
// 			}
// 		})
// 	}
// }

func (suite *KeeperTestSuite) TestQueryPacketCommitment() {
	var (
		expCommitment []byte
		path          *ibctesting.Path
		req           *types.QueryPacketCommitmentRequest
	)

	testCases := []struct {
		msg      string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				expCommitment = []byte("commitmentHash")
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), path.EndpointA.ClientID, 1, expCommitment)

				req = &types.QueryPacketCommitmentRequest{
					ClientId: path.EndpointA.ClientID,
					Sequence: 1,
				}
			},
			nil,
		},
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid channel ID",
			func() {
				req = &types.QueryPacketCommitmentRequest{
					ClientId: "",
					Sequence: 1,
				}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
		{
			"invalid sequence",
			func() {
				req = &types.QueryPacketCommitmentRequest{
					ClientId: ibctesting.FirstClientID,
					Sequence: 0,
				}
			},
			status.Error(codes.InvalidArgument, "packet sequence cannot be 0"),
		},
		{
			"commitment not found",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				req = &types.QueryPacketCommitmentRequest{
					ClientId: path.EndpointA.ClientID,
					Sequence: 1,
				}
			},
			status.Error(codes.NotFound, "packet commitment hash not found"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()

			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2)
			res, err := queryServer.PacketCommitment(suite.chainA.GetContext(), req)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expCommitment, res.Commitment)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Nil(res)
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
		expError error
	}{
		{
			"success",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				expCommitments = make([]*types.PacketState, 0, 10) // reset expected commitments
				for i := uint64(1); i <= 10; i++ {
					pktStateCommitment := types.NewPacketState(path.EndpointA.ClientID, i, []byte(fmt.Sprintf("hash_%d", i)))
					suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), pktStateCommitment.ClientId, pktStateCommitment.Sequence, pktStateCommitment.Data)
					expCommitments = append(expCommitments, &pktStateCommitment)
				}

				req = &types.QueryPacketCommitmentsRequest{
					ClientId: path.EndpointA.ClientID,
					Pagination: &query.PageRequest{
						Key:        nil,
						Limit:      11,
						CountTotal: true,
					},
				}
			},
			nil,
		},
		{
			"success: with pagination",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				expCommitments = make([]*types.PacketState, 0, 10) // reset expected commitments
				for i := uint64(1); i <= 10; i++ {
					pktStateCommitment := types.NewPacketState(path.EndpointA.ClientID, i, []byte(fmt.Sprintf("hash_%d", i)))
					suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), pktStateCommitment.ClientId, pktStateCommitment.Sequence, pktStateCommitment.Data)
					expCommitments = append(expCommitments, &pktStateCommitment)
				}

				limit := uint64(5)
				expCommitments = expCommitments[:limit]

				req = &types.QueryPacketCommitmentsRequest{
					ClientId: path.EndpointA.ClientID,
					Pagination: &query.PageRequest{
						Key:        nil,
						Limit:      limit,
						CountTotal: true,
					},
				}
			},
			nil,
		},
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid client ID",
			func() {
				req = &types.QueryPacketCommitmentsRequest{
					ClientId: "",
				}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2)
			res, err := queryServer.PacketCommitments(ctx, req)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expCommitments, res.Commitments)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryPacketAcknowledgement() {
	var (
		expAcknowledgement []byte
		path               *ibctesting.Path
		req                *types.QueryPacketAcknowledgementRequest
	)

	testCases := []struct {
		msg      string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				expAcknowledgement = []byte("acknowledgementHash")
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketAcknowledgement(suite.chainA.GetContext(), path.EndpointA.ClientID, 1, expAcknowledgement)

				req = &types.QueryPacketAcknowledgementRequest{
					ClientId: path.EndpointA.ClientID,
					Sequence: 1,
				}
			},
			nil,
		},
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid client ID",
			func() {
				req = &types.QueryPacketAcknowledgementRequest{
					ClientId: "",
					Sequence: 1,
				}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
		{
			"invalid sequence",
			func() {
				req = &types.QueryPacketAcknowledgementRequest{
					ClientId: ibctesting.FirstClientID,
					Sequence: 0,
				}
			},
			status.Error(codes.InvalidArgument, "packet sequence cannot be 0"),
		},
		{
			"acknowledgement not found",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				req = &types.QueryPacketAcknowledgementRequest{
					ClientId: path.EndpointA.ClientID,
					Sequence: 1,
				}
			},
			status.Error(codes.NotFound, "packet acknowledgement hash not found"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()

			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2)
			res, err := queryServer.PacketAcknowledgement(suite.chainA.GetContext(), req)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expAcknowledgement, res.Acknowledgement)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Nil(res)
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
		expError error
	}{
		{
			"success: with PacketCommitmentSequences",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				var commitments []uint64

				for i := uint64(0); i < 100; i++ {
					ack := types.NewPacketState(path.EndpointA.ClientID, i, []byte(fmt.Sprintf("hash_%d", i)))
					suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketAcknowledgement(suite.chainA.GetContext(), ack.ClientId, ack.Sequence, ack.Data)

					if i < 10 { // populate the store with 100 and query for 10 specific acks
						expAcknowledgements = append(expAcknowledgements, &ack)
						commitments = append(commitments, ack.Sequence)
					}
				}

				req = &types.QueryPacketAcknowledgementsRequest{
					ClientId:                  path.EndpointA.ClientID,
					PacketCommitmentSequences: commitments,
					Pagination:                nil,
				}
			},
			nil,
		},
		{
			"success: with pagination",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				expAcknowledgements = make([]*types.PacketState, 0, 10)

				for i := uint64(1); i <= 10; i++ {
					ack := types.NewPacketState(path.EndpointA.ClientID, i, []byte(fmt.Sprintf("hash_%d", i)))
					suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketAcknowledgement(suite.chainA.GetContext(), ack.ClientId, ack.Sequence, ack.Data)
					expAcknowledgements = append(expAcknowledgements, &ack)
				}

				req = &types.QueryPacketAcknowledgementsRequest{
					ClientId: path.EndpointA.ClientID,
					Pagination: &query.PageRequest{
						Key:        nil,
						Limit:      11,
						CountTotal: true,
					},
				}
			},
			nil,
		},
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
					ClientId: "",
				}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeperV2)
			res, err := queryServer.PacketAcknowledgements(ctx, req)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expAcknowledgements, res.Acknowledgements)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryPacketReceipt() {
	var (
		expReceipt bool
		path       *ibctesting.Path
		req        *types.QueryPacketReceiptRequest
	)

	testCases := []struct {
		msg      string
		malleate func()
		expError error
	}{
		{
			"success with receipt",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainA.GetContext(), path.EndpointA.ClientID, 1)

				expReceipt = true
				req = &types.QueryPacketReceiptRequest{
					ClientId: path.EndpointA.ClientID,
					Sequence: 1,
				}
			},
			nil,
		},
		{
			"success with no receipt",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				expReceipt = false
				req = &types.QueryPacketReceiptRequest{
					ClientId: path.EndpointA.ClientID,
					Sequence: 1,
				}
			},
			nil,
		},
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid client ID",
			func() {
				req = &types.QueryPacketReceiptRequest{
					ClientId: "",
					Sequence: 1,
				}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
		{
			"invalid sequence",
			func() {
				req = &types.QueryPacketReceiptRequest{
					ClientId: ibctesting.FirstClientID,
					Sequence: 0,
				}
			},
			status.Error(codes.InvalidArgument, "packet sequence cannot be 0"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()

			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2)
			res, err := queryServer.PacketReceipt(suite.chainA.GetContext(), req)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expReceipt, res.Received)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Nil(res)
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
		expError error
	}{
		{
			"success",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				expSeq = 42
				seq := uint64(42)
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetNextSequenceSend(suite.chainA.GetContext(), path.EndpointA.ClientID, seq)
				req = types.NewQueryNextSequenceSendRequest(path.EndpointA.ClientID)
			},
			nil,
		},
		{
			"req is nil",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid client ID",
			func() {
				req = types.NewQueryNextSequenceSendRequest("")
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
		{
			"sequence send not found",
			func() {
				req = types.NewQueryNextSequenceSendRequest(ibctesting.FirstClientID)
			},
			status.Error(codes.NotFound, fmt.Sprintf("client-id %s: sequence send not found", ibctesting.FirstClientID)),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeperV2)
			res, err := queryServer.NextSequenceSend(ctx, req)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expSeq, res.NextSequenceSend)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Nil(res)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryUnreceivedPackets() {
	var (
		expSeq []uint64
		path   *ibctesting.Path
		req    *types.QueryUnreceivedPacketsRequest
	)

	testCases := []struct {
		msg      string
		malleate func()
		expError error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid client ID",
			func() {
				req = &types.QueryUnreceivedPacketsRequest{
					ClientId: "",
				}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
		{
			"invalid seq",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				req = &types.QueryUnreceivedPacketsRequest{
					ClientId:  path.EndpointA.ClientID,
					Sequences: []uint64{0},
				}
			},
			status.Error(codes.InvalidArgument, "packet sequence 0 cannot be 0"),
		},
		{
			"basic success empty packet commitments",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				expSeq = []uint64(nil)
				req = &types.QueryUnreceivedPacketsRequest{
					ClientId:  path.EndpointA.ClientID,
					Sequences: []uint64{},
				}
			},
			nil,
		},
		{
			"basic success unreceived packet commitments",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				// no ack exists

				expSeq = []uint64{1}
				req = &types.QueryUnreceivedPacketsRequest{
					ClientId:  path.EndpointA.ClientID,
					Sequences: []uint64{1},
				}
			},
			nil,
		},
		{
			"basic success unreceived packet commitments, nothing to relay",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()

				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainA.GetContext(), path.EndpointA.ClientID, 1)

				expSeq = []uint64(nil)
				req = &types.QueryUnreceivedPacketsRequest{
					ClientId:  path.EndpointA.ClientID,
					Sequences: []uint64{1},
				}
			},
			nil,
		},
		{
			"success multiple unreceived packet commitments",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupV2()
				expSeq = []uint64(nil) // reset
				packetCommitments := []uint64{}

				// set packet receipt for every other sequence
				for seq := uint64(1); seq < 10; seq++ {
					packetCommitments = append(packetCommitments, seq)

					if seq%2 == 0 {
						suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainA.GetContext(), path.EndpointA.ClientID, seq)
					} else {
						expSeq = append(expSeq, seq)
					}
				}

				req = &types.QueryUnreceivedPacketsRequest{
					ClientId:  path.EndpointA.ClientID,
					Sequences: packetCommitments,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeperV2)
			res, err := queryServer.UnreceivedPackets(ctx, req)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expSeq, res.Sequences)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryUnreceivedAcks() {
	var (
		path   *ibctesting.Path
		req    *types.QueryUnreceivedAcksRequest
		expSeq = []uint64{}
	)

	testCases := []struct {
		msg      string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				expSeq = []uint64(nil)
				req = &types.QueryUnreceivedAcksRequest{
					ClientId:           path.EndpointA.ClientID,
					PacketAckSequences: []uint64{1},
				}
			},
			nil,
		},
		{
			"success: single unreceived packet ack",
			func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), path.EndpointA.ClientID, 1, []byte("commitment"))

				expSeq = []uint64{1}
				req = &types.QueryUnreceivedAcksRequest{
					ClientId:           path.EndpointA.ClientID,
					PacketAckSequences: []uint64{1},
				}
			},
			nil,
		},
		{
			"success: multiple unreceived packet acknowledgements",
			func() {
				expSeq = []uint64{} // reset
				packetAcks := []uint64{}

				// set packet commitment for every other sequence
				for seq := uint64(1); seq < 10; seq++ {
					packetAcks = append(packetAcks, seq)

					if seq%2 == 0 {
						suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), path.EndpointA.ClientID, seq, []byte("commitement"))
						expSeq = append(expSeq, seq)
					}
				}

				req = &types.QueryUnreceivedAcksRequest{
					ClientId:           path.EndpointA.ClientID,
					PacketAckSequences: packetAcks,
				}
			},
			nil,
		},
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid client ID",
			func() {
				req = &types.QueryUnreceivedAcksRequest{
					ClientId: "",
				}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
		{
			"invalid seq",
			func() {
				req = &types.QueryUnreceivedAcksRequest{
					ClientId:           path.EndpointA.ClientID,
					PacketAckSequences: []uint64{0},
				}
			},
			status.Error(codes.InvalidArgument, "packet sequence cannot be 0"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.App.GetIBCKeeper().ChannelKeeperV2)
			res, err := queryServer.UnreceivedAcks(ctx, req)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expSeq, res.Sequences)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Nil(res)
			}
		})
	}
}
