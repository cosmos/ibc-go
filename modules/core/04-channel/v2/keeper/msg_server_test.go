package keeper_test

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
	mockv2 "github.com/cosmos/ibc-go/v9/testing/mock/v2"
)

func (suite *KeeperTestSuite) TestMsgSendPacket() {
	var (
		path           *ibctesting.Path
		msg            *channeltypesv2.MsgSendPacket
		expectedPacket channeltypesv2.Packet
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name:     "success",
			malleate: func() {},
			expError: nil,
		},
		{
			name: "failure: timeout elapsed",
			malleate: func() {
				// ensure a message timeout.
				msg.TimeoutTimestamp = uint64(1)
			},
			expError: channeltypesv1.ErrTimeoutElapsed,
		},
		{
			name: "failure: inactive client",
			malleate: func() {
				path.EndpointA.FreezeClient()
			},
			expError: clienttypes.ErrClientNotActive,
		},
		{
			name: "failure: application callback error",
			malleate: func() {
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnSendPacket = func(ctx context.Context, sourceID string, destinationID string, sequence uint64, data channeltypesv2.PacketData, signer sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
			},
			expError: mock.MockApplicationCallbackError,
		},
		{
			name: "failure: counterparty not found",
			malleate: func() {
				msg.SourceChannel = "foo"
			},
			expError: channeltypesv1.ErrChannelNotFound,
		},
		{
			name: "failure: route to non existing app",
			malleate: func() {
				msg.PacketData[0].SourcePort = "foo"
			},
			expError: fmt.Errorf("no route for foo"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			timeoutTimestamp := suite.chainA.GetTimeoutTimestamp()
			msg = channeltypesv2.NewMsgSendPacket(path.EndpointA.ChannelID, timeoutTimestamp, suite.chainA.SenderAccount.GetAddress().String(), mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))

			expectedPacket = channeltypesv2.NewPacket(1, path.EndpointA.ChannelID, path.EndpointB.ChannelID, timeoutTimestamp, mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))

			tc.malleate()

			res, err := path.EndpointA.Chain.SendMsgs(msg)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				ck := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeperV2

				packetCommitment := ck.GetPacketCommitment(path.EndpointA.Chain.GetContext(), path.EndpointA.ChannelID, 1)
				suite.Require().NotNil(packetCommitment)
				suite.Require().Equal(channeltypesv2.CommitPacket(expectedPacket), packetCommitment, "packet commitment is not stored correctly")

				nextSequenceSend, ok := ck.GetNextSequenceSend(path.EndpointA.Chain.GetContext(), path.EndpointA.ChannelID)
				suite.Require().True(ok)
				suite.Require().Equal(uint64(2), nextSequenceSend, "next sequence send was not incremented correctly")

			} else {
				suite.Require().Error(err)
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgRecvPacket() {
	var (
		path        *ibctesting.Path
		msg         *channeltypesv2.MsgRecvPacket
		recvPacket  channeltypesv2.Packet
		expectedAck channeltypesv2.Acknowledgement
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name:     "success",
			malleate: func() {},
			expError: nil,
		},
		{
			name: "success: failed recv result",
			malleate: func() {
				failedRecvResult := channeltypesv2.RecvPacketResult{
					Status:          channeltypesv2.PacketStatus_Failure,
					Acknowledgement: mock.MockFailPacketData,
				}

				// a failed ack should be returned by the application.
				expectedAck.AcknowledgementResults[0].RecvPacketResult = failedRecvResult

				path.EndpointB.Chain.GetSimApp().MockModuleV2B.IBCApp.OnRecvPacket = func(ctx context.Context, sourceChannel string, destinationChannel string, data channeltypesv2.PacketData, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
					return failedRecvResult
				}
			},
		},
		{
			name: "success: async recv result",
			malleate: func() {
				asyncResult := channeltypesv2.RecvPacketResult{
					Status:          channeltypesv2.PacketStatus_Async,
					Acknowledgement: nil,
				}

				// an async ack should be returned by the application.
				expectedAck.AcknowledgementResults[0].RecvPacketResult = asyncResult

				path.EndpointB.Chain.GetSimApp().MockModuleV2B.IBCApp.OnRecvPacket = func(ctx context.Context, sourceChannel string, destinationChannel string, data channeltypesv2.PacketData, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
					return asyncResult
				}
			},
		},
		{
			name: "success: NoOp",
			malleate: func() {
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), recvPacket.SourceChannel, recvPacket.Sequence)
				expectedAck = channeltypesv2.Acknowledgement{}
			},
		},
		{
			name: "failure: counterparty not found",
			malleate: func() {
				// change the destination id to a non-existent channel.
				recvPacket.DestinationChannel = "not-existent-channel"
			},
			expError: channeltypesv2.ErrChannelNotFound,
		},
		{
			name: "failure: invalid proof",
			malleate: func() {
				// proof verification fails because the packet commitment is different due to a different sequence.
				recvPacket.Sequence = 10
			},
			expError: commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			timeoutTimestamp := suite.chainA.GetTimeoutTimestamp()
			msgSendPacket := channeltypesv2.NewMsgSendPacket(path.EndpointA.ChannelID, timeoutTimestamp, suite.chainA.SenderAccount.GetAddress().String(), mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))

			res, err := path.EndpointA.Chain.SendMsgs(msgSendPacket)
			suite.Require().NoError(err)
			suite.Require().NotNil(res)

			suite.Require().NoError(path.EndpointB.UpdateClient())

			recvPacket = channeltypesv2.NewPacket(1, path.EndpointA.ChannelID, path.EndpointB.ChannelID, timeoutTimestamp, mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))

			// default expected ack is a single successful recv result for moduleB.
			expectedAck = channeltypesv2.Acknowledgement{
				AcknowledgementResults: []channeltypesv2.AcknowledgementResult{
					{
						AppName: mockv2.ModuleNameB,
						RecvPacketResult: channeltypesv2.RecvPacketResult{
							Status:          channeltypesv2.PacketStatus_Success,
							Acknowledgement: mock.MockPacketData,
						},
					},
				},
			}

			tc.malleate()

			// get proof of packet commitment from chainA
			packetKey := hostv2.PacketCommitmentKey(recvPacket.SourceChannel, recvPacket.Sequence)
			proof, proofHeight := path.EndpointA.QueryProof(packetKey)

			msg = channeltypesv2.NewMsgRecvPacket(recvPacket, proof, proofHeight, suite.chainB.SenderAccount.GetAddress().String())

			res, err = path.EndpointB.Chain.SendMsgs(msg)
			suite.Require().NoError(path.EndpointA.UpdateClient())

			ck := path.EndpointB.Chain.GetSimApp().IBCKeeper.ChannelKeeperV2

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				// packet receipt should be written
				_, ok := ck.GetPacketReceipt(path.EndpointB.Chain.GetContext(), recvPacket.SourceChannel, recvPacket.Sequence)
				suite.Require().True(ok)

				ackWritten := ck.HasPacketAcknowledgement(path.EndpointB.Chain.GetContext(), recvPacket.DestinationChannel, recvPacket.Sequence)

				if len(expectedAck.AcknowledgementResults) == 0 || expectedAck.AcknowledgementResults[0].RecvPacketResult.Status == channeltypesv2.PacketStatus_Async {
					// ack should not be written for async app or if the packet receipt was already present.
					suite.Require().False(ackWritten)
				} else { // successful or failed acknowledgement
					// ack should be written for synchronous app (default mock application behaviour).
					suite.Require().True(ackWritten)
					expectedBz := channeltypesv2.CommitAcknowledgement(expectedAck)

					actualAckBz := ck.GetPacketAcknowledgement(path.EndpointB.Chain.GetContext(), recvPacket.DestinationChannel, recvPacket.Sequence)
					suite.Require().Equal(expectedBz, actualAckBz)
				}

			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError)
				_, ok := ck.GetPacketReceipt(path.EndpointB.Chain.GetContext(), recvPacket.SourceChannel, recvPacket.Sequence)
				suite.Require().False(ok)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestProvideCounterparty() {
	var (
		path *ibctesting.Path
		msg  *channeltypesv2.MsgProvideCounterparty
	)
	cases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				// set it before handler
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetChannel(suite.chainA.GetContext(), msg.ChannelId, channeltypesv2.NewChannel(path.EndpointA.ClientID, "", ibctesting.MerklePath))
			},
			nil,
		},
		{
			"failure: signer does not match creator",
			func() {
				msg.Signer = path.EndpointB.Chain.SenderAccount.GetAddress().String()
			},
			ibcerrors.ErrUnauthorized,
		},
		/* // Account sequence mismatch, expected 5, got 6. :thinking:
		{
			"failure: counterparty does not already exists",
			func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.ChannelStore(suite.chainA.GetContext(), path.EndpointA.ChannelID).Delete([]byte(channeltypesv2.ChannelKey))
			},
			channeltypesv2.ErrInvalidChannel,
		},
		*/
	}

	for _, tc := range cases {
		tc := tc
		path = ibctesting.NewPath(suite.chainA, suite.chainB)
		path.SetupClients()

		suite.Require().NoError(path.EndpointA.CreateChannel())
		suite.Require().NoError(path.EndpointB.CreateChannel())

		signer := path.EndpointA.Chain.SenderAccount.GetAddress().String()
		msg = channeltypesv2.NewMsgProvideCounterparty(path.EndpointA.ChannelID, path.EndpointB.ChannelID, signer)

		tc.malleate()

		res, err := path.EndpointA.Chain.SendMsgs(msg)

		expPass := tc.expError == nil
		if expPass {
			suite.Require().NotNil(res)
			suite.Require().Nil(err)

			// Assert counterparty channel id filled in and creator deleted
			channel, found := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelID)
			suite.Require().True(found)
			suite.Require().Equal(channel.CounterpartyChannelId, path.EndpointB.ChannelID)

			_, found = suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetCreator(suite.chainA.GetContext(), path.EndpointA.ChannelID)
			suite.Require().False(found)

			seq, found := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetNextSequenceSend(suite.chainA.GetContext(), path.EndpointA.ChannelID)
			suite.Require().True(found)
			suite.Require().Equal(seq, uint64(1))
		} else {
			suite.Require().Error(err)
		}
	}
}

func (suite *KeeperTestSuite) TestMsgAcknowledgement() {
	var (
		path         *ibctesting.Path
		msgAckPacket *channeltypesv2.MsgAcknowledgement
		recvPacket   channeltypesv2.Packet
	)
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name:     "success",
			malleate: func() {},
		},
		{
			name: "success: NoOp",
			malleate: func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), recvPacket.SourceChannel, recvPacket.Sequence, []byte{})

				// Modify the callback to return an error.
				// This way, we can verify that the callback is not executed in a No-op case.
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnAcknowledgementPacket = func(context.Context, string, string, channeltypesv2.PacketData, []byte, sdk.AccAddress) error {
					return errors.New("OnAcknowledgementPacket callback failed")
				}
			},
		},
		{
			name: "failure: invalid signer",
			malleate: func() {
				msgAckPacket.Signer = ""
			},
			expError: errors.New("empty address string is not allowed"),
		},
		{
			name: "failure: callback fails",
			malleate: func() {
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnAcknowledgementPacket = func(context.Context, string, string, channeltypesv2.PacketData, []byte, sdk.AccAddress) error {
					return errors.New("OnAcknowledgementPacket callback failed")
				}
			},
			expError: errors.New("OnAcknowledgementPacket callback failed"),
		},
		{
			name: "failure: counterparty not found",
			malleate: func() {
				// change the source id to a non-existent channel.
				msgAckPacket.Packet.SourceChannel = "not-existent-channel"
			},
			expError: channeltypesv2.ErrChannelNotFound,
		},
		{
			name: "failure: invalid commitment",
			malleate: func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), recvPacket.SourceChannel, recvPacket.Sequence, []byte("foo"))
			},
			expError: channeltypesv2.ErrInvalidPacket,
		},
		{
			name: "failure: failed membership verification",
			malleate: func() {
				msgAckPacket.ProofHeight = clienttypes.ZeroHeight()
			},
			expError: errors.New("failed packet acknowledgement verification"),
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			timeoutTimestamp := suite.chainA.GetTimeoutTimestamp()

			// Send packet from A to B
			msgSendPacket := channeltypesv2.NewMsgSendPacket(path.EndpointA.ChannelID, timeoutTimestamp, suite.chainA.SenderAccount.GetAddress().String(), mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))
			res, err := path.EndpointA.Chain.SendMsgs(msgSendPacket)
			suite.Require().NoError(err)
			suite.Require().NotNil(res)
			suite.Require().NoError(path.EndpointB.UpdateClient())

			// Receive packet on B
			recvPacket = channeltypesv2.NewPacket(1, path.EndpointA.ChannelID, path.EndpointB.ChannelID, timeoutTimestamp, mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))
			// get proof of packet commitment from chainA
			packetKey := hostv2.PacketCommitmentKey(recvPacket.SourceChannel, recvPacket.Sequence)
			proof, proofHeight := path.EndpointA.QueryProof(packetKey)

			// Construct msgRecvPacket to be sent to B
			msgRecvPacket := channeltypesv2.NewMsgRecvPacket(recvPacket, proof, proofHeight, suite.chainB.SenderAccount.GetAddress().String())
			res, err = suite.chainB.SendMsgs(msgRecvPacket)
			suite.Require().NoError(err)
			suite.Require().NotNil(res)
			suite.Require().NoError(path.EndpointA.UpdateClient())

			// Construct expected acknowledgement
			ack := channeltypesv2.Acknowledgement{
				AcknowledgementResults: []channeltypesv2.AcknowledgementResult{
					{
						AppName: mockv2.ModuleNameB,
						RecvPacketResult: channeltypesv2.RecvPacketResult{
							Status:          channeltypesv2.PacketStatus_Success,
							Acknowledgement: mock.MockPacketData,
						},
					},
				},
			}

			// Consttruct MsgAcknowledgement
			packetKey = hostv2.PacketAcknowledgementKey(recvPacket.DestinationChannel, recvPacket.Sequence)
			proof, proofHeight = path.EndpointB.QueryProof(packetKey)
			msgAckPacket = channeltypesv2.NewMsgAcknowledgement(recvPacket, ack, proof, proofHeight, suite.chainA.SenderAccount.GetAddress().String())

			tc.malleate()

			// Finally, acknowledge the packet on A
			res, err = suite.chainA.SendMsgs(msgAckPacket)

			expPass := tc.expError == nil

			if expPass {
				suite.Require().NoError(err)
				suite.NotNil(res)
			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError, "expected error %q, got %q instead", tc.expError, err)
			}
		})
	}
}
