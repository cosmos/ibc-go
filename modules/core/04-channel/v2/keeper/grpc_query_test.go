package keeper_test

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestQueryPacketCommitment() {
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
				path = ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupV2()

				expCommitment = []byte("commitmentHash")
				s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(s.chainA.GetContext(), path.EndpointA.ClientID, 1, expCommitment)

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
				path := ibctesting.NewPath(s.chainA, s.chainB)
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
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()

			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2)
			res, err := queryServer.PacketCommitment(s.chainA.GetContext(), req)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expCommitment, res.Commitment)
			} else {
				s.Require().ErrorIs(err, tc.expError)
				s.Require().Nil(res)
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
		expError error
	}{
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupV2()

				expCommitments = make([]*types.PacketState, 0, 10) // reset expected commitments
				for i := uint64(1); i <= 10; i++ {
					pktStateCommitment := types.NewPacketState(path.EndpointA.ClientID, i, fmt.Appendf(nil, "hash_%d", i))
					s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(s.chainA.GetContext(), pktStateCommitment.ClientId, pktStateCommitment.Sequence, pktStateCommitment.Data)
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
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupV2()

				expCommitments = make([]*types.PacketState, 0, 10) // reset expected commitments
				for i := uint64(1); i <= 10; i++ {
					pktStateCommitment := types.NewPacketState(path.EndpointA.ClientID, i, fmt.Appendf(nil, "hash_%d", i))
					s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(s.chainA.GetContext(), pktStateCommitment.ClientId, pktStateCommitment.Sequence, pktStateCommitment.Data)
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
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()

			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2)
			res, err := queryServer.PacketCommitments(ctx, req)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expCommitments, res.Commitments)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryPacketAcknowledgement() {
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
				path = ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupV2()

				expAcknowledgement = []byte("acknowledgementHash")
				s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketAcknowledgement(s.chainA.GetContext(), path.EndpointA.ClientID, 1, expAcknowledgement)

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
				path := ibctesting.NewPath(s.chainA, s.chainB)
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
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()

			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2)
			res, err := queryServer.PacketAcknowledgement(s.chainA.GetContext(), req)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expAcknowledgement, res.Acknowledgement)
			} else {
				s.Require().ErrorIs(err, tc.expError)
				s.Require().Nil(res)
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
		expError error
	}{
		{
			"success: with PacketCommitmentSequences",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupV2()

				var commitments []uint64

				for i := range uint64(100) {
					ack := types.NewPacketState(path.EndpointA.ClientID, i, fmt.Appendf(nil, "hash_%d", i))
					s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketAcknowledgement(s.chainA.GetContext(), ack.ClientId, ack.Sequence, ack.Data)

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
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupV2()

				expAcknowledgements = make([]*types.PacketState, 0, 10)

				for i := uint64(1); i <= 10; i++ {
					ack := types.NewPacketState(path.EndpointA.ClientID, i, fmt.Appendf(nil, "hash_%d", i))
					s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketAcknowledgement(s.chainA.GetContext(), ack.ClientId, ack.Sequence, ack.Data)
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
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()

			queryServer := keeper.NewQueryServer(s.chainA.App.GetIBCKeeper().ChannelKeeperV2)
			res, err := queryServer.PacketAcknowledgements(ctx, req)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expAcknowledgements, res.Acknowledgements)
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryPacketReceipt() {
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
				path = ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupV2()

				s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(s.chainA.GetContext(), path.EndpointA.ClientID, 1)

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
				path = ibctesting.NewPath(s.chainA, s.chainB)
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
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()

			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2)
			res, err := queryServer.PacketReceipt(s.chainA.GetContext(), req)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expReceipt, res.Received)
			} else {
				s.Require().ErrorIs(err, tc.expError)
				s.Require().Nil(res)
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
		expError error
	}{
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.Setup()

				expSeq = 42
				seq := uint64(42)
				s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetNextSequenceSend(s.chainA.GetContext(), path.EndpointA.ClientID, seq)
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
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()

			queryServer := keeper.NewQueryServer(s.chainA.App.GetIBCKeeper().ChannelKeeperV2)
			res, err := queryServer.NextSequenceSend(ctx, req)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expSeq, res.NextSequenceSend)
			} else {
				s.Require().ErrorIs(err, tc.expError)
				s.Require().Nil(res)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryUnreceivedPackets() {
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
				path := ibctesting.NewPath(s.chainA, s.chainB)
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
				path = ibctesting.NewPath(s.chainA, s.chainB)
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
				path = ibctesting.NewPath(s.chainA, s.chainB)
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
				path = ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupV2()

				s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(s.chainA.GetContext(), path.EndpointA.ClientID, 1)

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
				path = ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupV2()
				expSeq = []uint64(nil) // reset
				packetCommitments := []uint64{}

				// set packet receipt for every other sequence
				for seq := uint64(1); seq < 10; seq++ {
					packetCommitments = append(packetCommitments, seq)

					if seq%2 == 0 {
						s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(s.chainA.GetContext(), path.EndpointA.ClientID, seq)
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
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()

			queryServer := keeper.NewQueryServer(s.chainA.App.GetIBCKeeper().ChannelKeeperV2)
			res, err := queryServer.UnreceivedPackets(ctx, req)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expSeq, res.Sequences)
			} else {
				s.Require().ErrorIs(err, tc.expError)
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryUnreceivedAcks() {
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
				s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(s.chainA.GetContext(), path.EndpointA.ClientID, 1, []byte("commitment"))

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
						s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(s.chainA.GetContext(), path.EndpointA.ClientID, seq, []byte("commitement"))
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
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupV2()

			tc.malleate()
			ctx := s.chainA.GetContext()

			queryServer := keeper.NewQueryServer(s.chainA.App.GetIBCKeeper().ChannelKeeperV2)
			res, err := queryServer.UnreceivedAcks(ctx, req)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expSeq, res.Sequences)
			} else {
				s.Require().ErrorIs(err, tc.expError)
				s.Require().Nil(res)
			}
		})
	}
}
