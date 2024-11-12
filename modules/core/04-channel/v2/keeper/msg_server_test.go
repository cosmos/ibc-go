package keeper_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
	mockv2 "github.com/cosmos/ibc-go/v9/testing/mock/v2"
)

func (suite *KeeperTestSuite) TestRegisterCounterparty() {
	var (
		path *ibctesting.Path
		msg  *channeltypesv2.MsgRegisterCounterparty
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
			"failure: creator not set",
			func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeleteCreator(suite.chainA.GetContext(), path.EndpointA.ChannelID)
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: signer does not match creator",
			func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetCreator(suite.chainA.GetContext(), path.EndpointA.ChannelID, ibctesting.TestAccAddress)
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: channel must already exist",
			func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeleteCreator(suite.chainA.GetContext(), path.EndpointA.ChannelID)
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.ChannelStore(suite.chainA.GetContext(), path.EndpointA.ChannelID).Delete([]byte(channeltypesv2.ChannelKey))
			},
			channeltypesv2.ErrChannelNotFound,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()

			suite.Require().NoError(path.EndpointA.CreateChannel())
			suite.Require().NoError(path.EndpointB.CreateChannel())

			signer := path.EndpointA.Chain.SenderAccount.GetAddress().String()
			msg = channeltypesv2.NewMsgRegisterCounterparty(path.EndpointA.ChannelID, path.EndpointB.ChannelID, signer)

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
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError, "expected error %q, got %q instead", tc.expError, err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgSendPacket() {
	var (
		path             *ibctesting.Path
		expectedPacket   channeltypesv2.Packet
		timeoutTimestamp uint64
		payload          channeltypesv2.Payload
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
			name: "success: valid timeout timestamp",
			malleate: func() {
				// ensure a message timeout.
				timeoutTimestamp = uint64(suite.chainA.GetContext().BlockTime().Add(channeltypesv2.MaxTimeoutDelta - 10*time.Second).Unix())
				expectedPacket = channeltypesv2.NewPacket(1, path.EndpointA.ChannelID, path.EndpointB.ChannelID, timeoutTimestamp, payload)
			},
			expError: nil,
		},
		{
			name: "failure: timeout elapsed",
			malleate: func() {
				// ensure a message timeout.
				timeoutTimestamp = uint64(1)
			},
			expError: channeltypesv2.ErrTimeoutElapsed,
		},
		{
			name: "failure: timeout timestamp exceeds max allowed input",
			malleate: func() {
				// ensure message timeout exceeds max allowed input.
				timeoutTimestamp = uint64(suite.chainA.GetContext().BlockTime().Add(channeltypesv2.MaxTimeoutDelta + 10*time.Second).Unix())
			},
			expError: channeltypesv2.ErrInvalidTimeout,
		},
		{
			name: "failure: timeout timestamp less than current block timestamp",
			malleate: func() {
				// ensure message timeout exceeds max allowed input.
				timeoutTimestamp = uint64(suite.chainA.GetContext().BlockTime().Unix()) - 1
			},
			expError: channeltypesv2.ErrTimeoutElapsed,
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
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnSendPacket = func(ctx context.Context, sourceID string, destinationID string, sequence uint64, data channeltypesv2.Payload, signer sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
			},
			expError: mock.MockApplicationCallbackError,
		},
		{
			name: "failure: channel not found",
			malleate: func() {
				path.EndpointA.ChannelID = ibctesting.InvalidID
			},
			expError: channeltypesv2.ErrChannelNotFound,
		},
		{
			name: "failure: route to non existing app",
			malleate: func() {
				payload.SourcePort = "foo"
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

			timeoutTimestamp = suite.chainA.GetTimeoutTimestampSecs()
			payload = mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			expectedPacket = channeltypesv2.NewPacket(1, path.EndpointA.ChannelID, path.EndpointB.ChannelID, timeoutTimestamp, payload)

			tc.malleate()

			packet, err := path.EndpointA.MsgSendPacket(timeoutTimestamp, payload)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(packet)

				ck := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeperV2

				packetCommitment := ck.GetPacketCommitment(path.EndpointA.Chain.GetContext(), path.EndpointA.ChannelID, 1)
				suite.Require().NotNil(packetCommitment)
				suite.Require().Equal(channeltypesv2.CommitPacket(expectedPacket), packetCommitment, "packet commitment is not stored correctly")

				nextSequenceSend, ok := ck.GetNextSequenceSend(path.EndpointA.Chain.GetContext(), path.EndpointA.ChannelID)
				suite.Require().True(ok)
				suite.Require().Equal(uint64(2), nextSequenceSend, "next sequence send was not incremented correctly")

				suite.Require().Equal(expectedPacket, packet)

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
		packet      channeltypesv2.Packet
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

				path.EndpointB.Chain.GetSimApp().MockModuleV2B.IBCApp.OnRecvPacket = func(ctx context.Context, sourceChannel string, destinationChannel string, data channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
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

				path.EndpointB.Chain.GetSimApp().MockModuleV2B.IBCApp.OnRecvPacket = func(ctx context.Context, sourceChannel string, destinationChannel string, data channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
					return asyncResult
				}
			},
		},
		{
			name: "success: NoOp",
			malleate: func() {
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationChannel, packet.Sequence)
				expectedAck = channeltypesv2.Acknowledgement{}
			},
		},
		{
			name: "failure: counterparty not found",
			malleate: func() {
				// change the destination id to a non-existent channel.
				packet.DestinationChannel = ibctesting.InvalidID
			},
			expError: channeltypesv2.ErrChannelNotFound,
		},
		{
			name: "failure: invalid proof",
			malleate: func() {
				// proof verification fails because the packet commitment is different due to a different sequence.
				packet.Sequence = 10
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

			timeoutTimestamp := suite.chainA.GetTimeoutTimestampSecs()

			var err error
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB))
			suite.Require().NoError(err)

			// default expected ack is a single successful recv result for moduleB.
			expectedAck = channeltypesv2.Acknowledgement{
				AcknowledgementResults: []channeltypesv2.AcknowledgementResult{
					{
						AppName:          mockv2.ModuleNameB,
						RecvPacketResult: mockv2.MockRecvPacketResult,
					},
				},
			}

			tc.malleate()

			err = path.EndpointB.MsgRecvPacket(packet)

			ck := path.EndpointB.Chain.GetSimApp().IBCKeeper.ChannelKeeperV2

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				// packet receipt should be written
				_, ok := ck.GetPacketReceipt(path.EndpointB.Chain.GetContext(), packet.DestinationChannel, packet.Sequence)
				suite.Require().True(ok)

				ackWritten := ck.HasPacketAcknowledgement(path.EndpointB.Chain.GetContext(), packet.DestinationChannel, packet.Sequence)

				if len(expectedAck.AcknowledgementResults) == 0 || expectedAck.AcknowledgementResults[0].RecvPacketResult.Status == channeltypesv2.PacketStatus_Async {
					// ack should not be written for async app or if the packet receipt was already present.
					suite.Require().False(ackWritten)
				} else { // successful or failed acknowledgement
					// ack should be written for synchronous app (default mock application behaviour).
					suite.Require().True(ackWritten)
					expectedBz := channeltypesv2.CommitAcknowledgement(expectedAck)

					actualAckBz := ck.GetPacketAcknowledgement(path.EndpointB.Chain.GetContext(), packet.DestinationChannel, packet.Sequence)
					suite.Require().Equal(expectedBz, actualAckBz)
				}

			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError)
				_, ok := ck.GetPacketReceipt(path.EndpointB.Chain.GetContext(), packet.SourceChannel, packet.Sequence)
				suite.Require().False(ok)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgAcknowledgement() {
	var (
		path   *ibctesting.Path
		packet channeltypesv2.Packet
		ack    channeltypesv2.Acknowledgement
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
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeletePacketCommitment(suite.chainA.GetContext(), packet.SourceChannel, packet.Sequence)

				// Modify the callback to return an error.
				// This way, we can verify that the callback is not executed in a No-op case.
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnAcknowledgementPacket = func(context.Context, string, string, channeltypesv2.Payload, []byte, sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
			},
		},
		{
			name: "failure: callback fails",
			malleate: func() {
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnAcknowledgementPacket = func(context.Context, string, string, channeltypesv2.Payload, []byte, sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
			},
			expError: mock.MockApplicationCallbackError,
		},
		{
			name: "failure: counterparty not found",
			malleate: func() {
				// change the source id to a non-existent channel.
				packet.SourceChannel = "not-existent-channel"
			},
			expError: channeltypesv2.ErrChannelNotFound,
		},
		{
			name: "failure: invalid commitment",
			malleate: func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), packet.SourceChannel, packet.Sequence, []byte("foo"))
			},
			expError: channeltypesv2.ErrInvalidPacket,
		},
		{
			name: "failure: failed membership verification",
			malleate: func() {
				ack.AcknowledgementResults[0].RecvPacketResult.Acknowledgement = mock.MockFailPacketData
			},
			expError: errors.New("failed packet acknowledgement verification"),
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			timeoutTimestamp := suite.chainA.GetTimeoutTimestampSecs()

			var err error
			// Send packet from A to B
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB))
			suite.Require().NoError(err)

			err = path.EndpointB.MsgRecvPacket(packet)
			suite.Require().NoError(err)

			// Construct expected acknowledgement
			ack = channeltypesv2.Acknowledgement{
				AcknowledgementResults: []channeltypesv2.AcknowledgementResult{
					{
						AppName:          mockv2.ModuleNameB,
						RecvPacketResult: mockv2.MockRecvPacketResult,
					},
				},
			}

			tc.malleate()

			// Finally, acknowledge the packet on A
			err = path.EndpointA.MsgAcknowledgePacket(packet, ack)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError, "expected error %q, got %q instead", tc.expError, err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgTimeout() {
	var (
		path   *ibctesting.Path
		packet channeltypesv2.Packet
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
			name: "failure: no-op",
			malleate: func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeletePacketCommitment(suite.chainA.GetContext(), packet.SourceChannel, packet.Sequence)

				// Modify the callback to return a different error.
				// This way, we can verify that the callback is not executed in a No-op case.
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnTimeoutPacket = func(context.Context, string, string, channeltypesv2.Payload, sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
			},
			expError: channeltypesv2.ErrNoOpMsg,
		},
		{
			name: "failure: callback fails",
			malleate: func() {
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnTimeoutPacket = func(context.Context, string, string, channeltypesv2.Payload, sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
			},
			expError: mock.MockApplicationCallbackError,
		},
		{
			name: "failure: channel not found",
			malleate: func() {
				// change the source id to a non-existent channel.
				packet.SourceChannel = "not-existent-channel"
			},
			expError: channeltypesv2.ErrChannelNotFound,
		},
		{
			name: "failure: invalid commitment",
			malleate: func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), packet.SourceChannel, packet.Sequence, []byte("foo"))
			},
			expError: channeltypesv2.ErrInvalidPacket,
		},
		{
			name: "failure: unable to timeout if packet has been received",
			malleate: func() {
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.SourceChannel, packet.Sequence)
				suite.Require().NoError(path.EndpointB.UpdateClient())
			},
			expError: commitmenttypes.ErrInvalidProof,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			// Send packet from A to B
			timeoutTimestamp := uint64(suite.chainA.GetContext().BlockTime().Unix())
			mockData := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			var err error
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, mockData)
			suite.Require().NoError(err)
			suite.Require().NotEmpty(packet)

			tc.malleate()

			suite.Require().NoError(path.EndpointA.UpdateClient())

			err = path.EndpointA.MsgTimeoutPacket(packet)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError, "expected error %q, got %q instead", tc.expError, err)
			}
		})
	}
}
