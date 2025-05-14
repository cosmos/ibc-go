package keeper_test

import (
	"errors"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/cosmos/ibc-go/v10/testing/mock"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
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
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnSendPacket = func(ctx sdk.Context, sourceID string, destinationID string, sequence uint64, data types.Payload, signer sdk.AccAddress) error {
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
			expError: clientv2types.ErrCounterpartyNotFound,
		},
		{
			name: "failure: route to non existing app",
			malleate: func() {
				payload.SourcePort = "foo"
			},
			expError: errors.New("no route for foo"),
		},
	}

	for _, tc := range testCases {
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
			name: "success: receive permissioned with msg sender",
			malleate: func() {
				creator := suite.chainB.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointB.ClientID, creator.String(), clientv2types.NewConfig(suite.chainA.SenderAccount.GetAddress().String(), creator.String()))
				_, err := suite.chainB.App.GetIBCKeeper().UpdateClientConfig(suite.chainB.GetContext(), msg)
				suite.Require().NoError(err)
			},
			expError:      nil,
			expAckWritten: true,
		},
		{
			name: "failure: relayer not permissioned",
			malleate: func() {
				creator := suite.chainB.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointB.ClientID, creator.String(), clientv2types.NewConfig(suite.chainA.SenderAccount.GetAddress().String()))
				_, err := suite.chainB.App.GetIBCKeeper().UpdateClientConfig(suite.chainB.GetContext(), msg)
				suite.Require().NoError(err)
			},
			expError: ibcerrors.ErrUnauthorized,
		},
		{
			name: "failure: counterparty not found",
			malleate: func() {
				// change the destination id to a non-existent channel.
				packet.DestinationClient = ibctesting.InvalidID
			},
			expError: clientv2types.ErrCounterpartyNotFound,
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
			path.EndpointB.Chain.GetSimApp().MockModuleV2B.IBCApp.OnRecvPacket = func(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, data types.Payload, relayer sdk.AccAddress) types.RecvPacketResult {
				return expRecvRes
			}

			err = path.EndpointB.MsgRecvPacket(packet)
			suite.Require().NoError(err)

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
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnAcknowledgementPacket = func(sdk.Context, string, string, uint64, types.Payload, []byte, sdk.AccAddress) error {
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
			name: "success: relayer permissioned with msg sender",
			malleate: func() {
				creator := suite.chainA.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, creator.String(), clientv2types.NewConfig(suite.chainB.SenderAccount.GetAddress().String(), creator.String()))
				_, err := suite.chainA.App.GetIBCKeeper().UpdateClientConfig(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)
			},
			payload: mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
		},
		{
			name: "failure: relayer not permissioned",
			malleate: func() {
				creator := suite.chainA.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, creator.String(), clientv2types.NewConfig(suite.chainB.SenderAccount.GetAddress().String()))
				_, err := suite.chainA.App.GetIBCKeeper().UpdateClientConfig(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)
			},
			payload:  mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
			expError: ibcerrors.ErrUnauthorized,
		},
		{
			name: "failure: callback fails",
			malleate: func() {
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnAcknowledgementPacket = func(sdk.Context, string, string, uint64, types.Payload, []byte, sdk.AccAddress) error {
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
			expError: clientv2types.ErrCounterpartyNotFound,
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
			name: "success",
			malleate: func() {
				suite.Require().NoError(path.EndpointA.UpdateClient())
			},
		},
		{
			name: "success: no-op",
			malleate: func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeletePacketCommitment(suite.chainA.GetContext(), packet.SourceClient, packet.Sequence)

				// Modify the callback to return a different error.
				// This way, we can verify that the callback is not executed in a No-op case.
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnTimeoutPacket = func(sdk.Context, string, string, uint64, types.Payload, sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
				suite.Require().NoError(path.EndpointA.UpdateClient())
			},
		},
		{
			name: "success: relayer permissioned with msg sender",
			malleate: func() {
				creator := suite.chainA.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, creator.String(), clientv2types.NewConfig(suite.chainB.SenderAccount.GetAddress().String(), creator.String()))
				_, err := suite.chainA.App.GetIBCKeeper().UpdateClientConfig(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)
				suite.Require().NoError(path.EndpointA.UpdateClient())
			},
		},
		{
			name: "failure: relayer not permissioned",
			malleate: func() {
				// update first before permissioning the relayer in this case
				suite.Require().NoError(path.EndpointA.UpdateClient())
				creator := suite.chainA.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, creator.String(), clientv2types.NewConfig(suite.chainB.SenderAccount.GetAddress().String()))
				_, err := suite.chainA.App.GetIBCKeeper().UpdateClientConfig(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)
			},
			expError: ibcerrors.ErrUnauthorized,
		},
		{
			name: "failure: callback fails",
			malleate: func() {
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnTimeoutPacket = func(sdk.Context, string, string, uint64, types.Payload, sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
				suite.Require().NoError(path.EndpointA.UpdateClient())
			},
			expError: mock.MockApplicationCallbackError,
		},
		{
			name: "failure: client not found",
			malleate: func() {
				// change the source id to a non-existent client.
				packet.SourceClient = "not-existent-client"
				suite.Require().NoError(path.EndpointA.UpdateClient())
			},
			expError: clientv2types.ErrCounterpartyNotFound,
		},
		{
			name: "failure: invalid commitment",
			malleate: func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), packet.SourceClient, packet.Sequence, []byte("foo"))
				suite.Require().NoError(path.EndpointA.UpdateClient())
			},
			expError: types.ErrInvalidPacket,
		},
		{
			name: "failure: unable to timeout if packet has been received",
			malleate: func() {
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
				suite.Require().NoError(path.EndpointB.UpdateClient())
				suite.Require().NoError(path.EndpointA.UpdateClient())
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
			// make timeoutTimestamp 1 second more than sending chain time to ensure it passes SendPacket
			// and times out successfully after update
			timeoutTimestamp := uint64(suite.chainA.GetContext().BlockTime().Add(time.Second).Unix())
			mockData := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			var err error
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, mockData)
			suite.Require().NoError(err)
			suite.Require().NotEmpty(packet)

			tc.malleate()

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
