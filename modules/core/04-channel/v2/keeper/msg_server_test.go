package keeper_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
	mockv2 "github.com/cosmos/ibc-go/v9/testing/mock/v2"
)

func (suite *KeeperTestSuite) TestMsgSendPacket() {
	var (
		path             *ibctesting.Path
		expectedPacket   types.Packet
		timeoutTimestamp uint64
		payload          types.Payload
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
				timeoutTimestamp = uint64(suite.chainA.GetContext().BlockTime().Add(types.MaxTimeoutDelta - 10*time.Second).Unix())
				expectedPacket = types.NewPacket(1, path.EndpointA.ClientID, path.EndpointB.ClientID, timeoutTimestamp, payload)
			},
			expError: nil,
		},
		{
			name: "failure: timeout elapsed",
			malleate: func() {
				// ensure a message timeout.
				timeoutTimestamp = uint64(1)
			},
			expError: types.ErrTimeoutElapsed,
		},
		{
			name: "failure: timeout timestamp exceeds max allowed input",
			malleate: func() {
				// ensure message timeout exceeds max allowed input.
				timeoutTimestamp = uint64(suite.chainA.GetContext().BlockTime().Add(types.MaxTimeoutDelta + 10*time.Second).Unix())
			},
			expError: types.ErrInvalidTimeout,
		},
		{
			name: "failure: timeout timestamp less than current block timestamp",
			malleate: func() {
				// ensure message timeout exceeds max allowed input.
				timeoutTimestamp = uint64(suite.chainA.GetContext().BlockTime().Unix()) - 1
			},
			expError: types.ErrTimeoutElapsed,
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
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnSendPacket = func(ctx context.Context, sourceID string, destinationID string, sequence uint64, data types.Payload, signer sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
			},
			expError: mock.MockApplicationCallbackError,
		},
		{
			name: "failure: client not found",
			malleate: func() {
				path.EndpointA.ClientID = ibctesting.InvalidID
			},
			expError: clienttypes.ErrCounterpartyNotFound,
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

			expectedPacket = types.NewPacket(1, path.EndpointA.ClientID, path.EndpointB.ClientID, timeoutTimestamp, payload)

			tc.malleate()

			packet, err := path.EndpointA.MsgSendPacket(timeoutTimestamp, payload)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(packet)

				ck := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeperV2

				packetCommitment := ck.GetPacketCommitment(path.EndpointA.Chain.GetContext(), path.EndpointA.ClientID, 1)
				suite.Require().NotNil(packetCommitment)
				suite.Require().Equal(types.CommitPacket(expectedPacket), packetCommitment, "packet commitment is not stored correctly")

				nextSequenceSend, ok := ck.GetNextSequenceSend(path.EndpointA.Chain.GetContext(), path.EndpointA.ClientID)
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
		path       *ibctesting.Path
		packet     types.Packet
		expRecvRes types.RecvPacketResult
	)

	testCases := []struct {
		name          string
		malleate      func()
		expError      error
		expAckWritten bool
	}{
		{
			name:          "success",
			malleate:      func() {},
			expError:      nil,
			expAckWritten: true,
		},
		{
			name: "success: failed recv result",
			malleate: func() {
				expRecvRes = types.RecvPacketResult{
					Status: types.PacketStatus_Failure,
				}
			},
			expError:      nil,
			expAckWritten: true,
		},
		{
			name: "success: async recv result",
			malleate: func() {
				expRecvRes = types.RecvPacketResult{
					Status: types.PacketStatus_Async,
				}
			},
			expError:      nil,
			expAckWritten: false,
		},
		{
			name: "success: NoOp",
			malleate: func() {
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
			},
			expError:      nil,
			expAckWritten: false,
		},
		{
			name: "failure: counterparty not found",
			malleate: func() {
				// change the destination id to a non-existent channel.
				packet.DestinationClient = ibctesting.InvalidID
			},
			expError: clienttypes.ErrCounterpartyNotFound,
		},
		{
			name: "failure: invalid proof",
			malleate: func() {
				// proof verification fails because the packet commitment is different due to a different sequence.
				packet.Sequence = 10
			},
			expError: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "failure: invalid acknowledgement",
			malleate: func() {
				expRecvRes = types.RecvPacketResult{
					Status:          types.PacketStatus_Success,
					Acknowledgement: []byte(""),
				}
			},
			expError: types.ErrInvalidAcknowledgement,
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

			// default expected receive result is a single successful recv result for moduleB.
			expRecvRes = mockv2.MockRecvPacketResult

			tc.malleate()

			// expectedAck is derived from the expected recv result.
			var expectedAck types.Acknowledgement
			if expRecvRes.Status == types.PacketStatus_Success {
				expectedAck = types.Acknowledgement{AppAcknowledgements: [][]byte{expRecvRes.Acknowledgement}}
			} else {
				expectedAck = types.Acknowledgement{AppAcknowledgements: [][]byte{types.ErrorAcknowledgement[:]}}
			}

			// modify the callback to return the expected recv result.
			path.EndpointB.Chain.GetSimApp().MockModuleV2B.IBCApp.OnRecvPacket = func(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, data types.Payload, relayer sdk.AccAddress) types.RecvPacketResult {
				return expRecvRes
			}

			err = path.EndpointB.MsgRecvPacket(packet)

			ck := path.EndpointB.Chain.GetSimApp().IBCKeeper.ChannelKeeperV2

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				// packet receipt should be written
				_, ok := ck.GetPacketReceipt(path.EndpointB.Chain.GetContext(), packet.DestinationClient, packet.Sequence)
				suite.Require().True(ok)

				ackWritten := ck.HasPacketAcknowledgement(path.EndpointB.Chain.GetContext(), packet.DestinationClient, packet.Sequence)

				if !tc.expAckWritten {
					// ack should not be written for async app or if the packet receipt was already present.
					suite.Require().False(ackWritten)
				} else { // successful or failed acknowledgement
					// ack should be written for synchronous app (default mock application behaviour).
					suite.Require().True(ackWritten)
					expectedBz := types.CommitAcknowledgement(expectedAck)

					actualAckBz := ck.GetPacketAcknowledgement(path.EndpointB.Chain.GetContext(), packet.DestinationClient, packet.Sequence)
					suite.Require().Equal(expectedBz, actualAckBz)
				}

			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError)
				_, ok := ck.GetPacketReceipt(path.EndpointB.Chain.GetContext(), packet.SourceClient, packet.Sequence)
				suite.Require().False(ok)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgAcknowledgement() {
	var (
		path   *ibctesting.Path
		packet types.Packet
		ack    types.Acknowledgement
	)
	testCases := []struct {
		name     string
		malleate func()
		payload  types.Payload
		expError error
	}{
		{
			name:     "success",
			malleate: func() {},
			payload:  mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
		},
		{
			name: "success: NoOp",
			malleate: func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeletePacketCommitment(suite.chainA.GetContext(), packet.SourceClient, packet.Sequence)

				// Modify the callback to return an error.
				// This way, we can verify that the callback is not executed in a No-op case.
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnAcknowledgementPacket = func(context.Context, string, string, uint64, types.Payload, []byte, sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
			},
			payload: mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
		},
		{
			name: "success: failed ack result",
			malleate: func() {
				ack.AppAcknowledgements[0] = types.ErrorAcknowledgement[:]
			},
			payload: mockv2.NewErrorMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
		},
		{
			name: "failure: callback fails",
			malleate: func() {
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnAcknowledgementPacket = func(context.Context, string, string, uint64, types.Payload, []byte, sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
			},
			payload:  mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
			expError: mock.MockApplicationCallbackError,
		},
		{
			name: "failure: counterparty not found",
			malleate: func() {
				// change the source id to a non-existent channel.
				packet.SourceClient = "not-existent-channel"
			},
			payload:  mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
			expError: clienttypes.ErrCounterpartyNotFound,
		},
		{
			name: "failure: invalid commitment",
			malleate: func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), packet.SourceClient, packet.Sequence, []byte("foo"))
			},
			payload:  mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
			expError: types.ErrInvalidPacket,
		},
		{
			name: "failure: failed membership verification",
			malleate: func() {
				ack.AppAcknowledgements[0] = mock.MockFailPacketData
			},
			payload:  mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
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
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, tc.payload)
			suite.Require().NoError(err)

			err = path.EndpointB.MsgRecvPacket(packet)
			suite.Require().NoError(err)

			// Construct expected acknowledgement
			ack = types.Acknowledgement{AppAcknowledgements: [][]byte{mockv2.MockRecvPacketResult.Acknowledgement}}

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
		packet types.Packet
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
			name: "success: no-op",
			malleate: func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeletePacketCommitment(suite.chainA.GetContext(), packet.SourceClient, packet.Sequence)

				// Modify the callback to return a different error.
				// This way, we can verify that the callback is not executed in a No-op case.
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnTimeoutPacket = func(context.Context, string, string, uint64, types.Payload, sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
			},
		},
		{
			name: "failure: callback fails",
			malleate: func() {
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnTimeoutPacket = func(context.Context, string, string, uint64, types.Payload, sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
			},
			expError: mock.MockApplicationCallbackError,
		},
		{
			name: "failure: client not found",
			malleate: func() {
				// change the source id to a non-existent client.
				packet.SourceClient = "not-existent-client"
			},
			expError: clienttypes.ErrCounterpartyNotFound,
		},
		{
			name: "failure: invalid commitment",
			malleate: func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), packet.SourceClient, packet.Sequence, []byte("foo"))
			},
			expError: types.ErrInvalidPacket,
		},
		{
			name: "failure: unable to timeout if packet has been received",
			malleate: func() {
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
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
