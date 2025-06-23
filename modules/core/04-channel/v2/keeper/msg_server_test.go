package keeper_test

import (
	"bytes"
	"errors"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	mockv1 "github.com/cosmos/ibc-go/v10/testing/mock"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
)

func (s *KeeperTestSuite) TestMsgSendPacket() {
	var (
		path             *ibctesting.Path
		expectedPacket   types.Packet
		timeoutTimestamp uint64
		payloads         []types.Payload
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
			name: "success multiple payloads",
			malleate: func() {
				payloads = append(payloads, payloads[0])
			},
			expError: nil,
		},
		{
			name: "success: valid timeout timestamp",
			malleate: func() {
				// ensure a message timeout.
				timeoutTimestamp = uint64(s.chainA.GetContext().BlockTime().Add(types.MaxTimeoutDelta - 10*time.Second).Unix())
				expectedPacket = types.NewPacket(1, path.EndpointA.ClientID, path.EndpointB.ClientID, timeoutTimestamp, payloads...)
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
				timeoutTimestamp = uint64(s.chainA.GetContext().BlockTime().Add(types.MaxTimeoutDelta + 10*time.Second).Unix())
			},
			expError: types.ErrInvalidTimeout,
		},
		{
			name: "failure: timeout timestamp less than current block timestamp",
			malleate: func() {
				// ensure message timeout exceeds max allowed input.
				timeoutTimestamp = uint64(s.chainA.GetContext().BlockTime().Unix()) - 1
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
					return mockv1.MockApplicationCallbackError
				}
			},
			expError: mockv1.MockApplicationCallbackError,
		},
		{
			name: "failure: multiple payload application callback error",
			malleate: func() {
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnSendPacket = func(ctx sdk.Context, sourceID string, destinationID string, sequence uint64, data types.Payload, signer sdk.AccAddress) error {
					if bytes.Equal(mockv1.MockFailPacketData, data.Value) {
						return mockv1.MockApplicationCallbackError
					}
					return nil
				}
				payloads = append(payloads, mockv2.NewErrorMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB))
			},
			expError: mockv1.MockApplicationCallbackError,
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
				payloads[0].SourcePort = "foo"
			},
			expError: errors.New("no route for foo"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupV2()

			timeoutTimestamp = s.chainA.GetTimeoutTimestampSecs()
			payloads = []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)}

			tc.malleate()

			expectedPacket = types.NewPacket(1, path.EndpointA.ClientID, path.EndpointB.ClientID, timeoutTimestamp, payloads...)
			packet, err := path.EndpointA.MsgSendPacket(timeoutTimestamp, payloads...)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
				s.Require().NotEmpty(packet)

				ck := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeperV2

				packetCommitment := ck.GetPacketCommitment(path.EndpointA.Chain.GetContext(), path.EndpointA.ClientID, 1)
				s.Require().NotNil(packetCommitment)
				s.Require().Equal(types.CommitPacket(expectedPacket), packetCommitment, "packet commitment is not stored correctly")

				nextSequenceSend, ok := ck.GetNextSequenceSend(path.EndpointA.Chain.GetContext(), path.EndpointA.ClientID)
				s.Require().True(ok)
				s.Require().Equal(uint64(2), nextSequenceSend, "next sequence send was not incremented correctly")

				s.Require().Equal(expectedPacket, packet)
			} else {
				s.Require().Error(err)
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expError)
			}
		})
	}
}

func (s *KeeperTestSuite) TestMsgRecvPacket() {
	var (
		path   *ibctesting.Path
		packet types.Packet
		expAck types.Acknowledgement
	)

	testCases := []struct {
		name          string
		payloads      []types.Payload
		malleate      func()
		expError      error
		expAckWritten bool
	}{
		{
			name:          "success",
			payloads:      []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			malleate:      func() {},
			expError:      nil,
			expAckWritten: true,
		},
		{
			name:     "success: error ack",
			payloads: []types.Payload{mockv2.NewErrorMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			malleate: func() {
				expAck = types.Acknowledgement{
					AppAcknowledgements: [][]byte{types.ErrorAcknowledgement[:]},
				}
			},
			expError:      nil,
			expAckWritten: true,
		},
		{
			name:          "success: async recv result",
			payloads:      []types.Payload{mockv2.NewAsyncMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			malleate:      func() {},
			expError:      nil,
			expAckWritten: false,
		},
		{
			name:     "success: NoOp",
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			malleate: func() {
				s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
			},
			expError:      nil,
			expAckWritten: false,
		},
		{
			name: "success: multiple payloads",
			payloads: []types.Payload{
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
			},
			malleate: func() {
				expAck = types.Acknowledgement{
					AppAcknowledgements: [][]byte{
						mockv2.MockRecvPacketResult.Acknowledgement,
						mockv2.MockRecvPacketResult.Acknowledgement,
					},
				}
			},
			expError:      nil,
			expAckWritten: true,
		},
		{
			name: "success: multiple payloads with error ack",
			payloads: []types.Payload{
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
				mockv2.NewErrorMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
			},
			malleate: func() {
				expAck = types.Acknowledgement{
					AppAcknowledgements: [][]byte{
						types.ErrorAcknowledgement[:],
					},
				}
			},
			expError:      nil,
			expAckWritten: true,
		},
		{
			name:     "success: receive permissioned with msg sender",
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},

			malleate: func() {
				creator := s.chainB.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointB.ClientID, creator.String(), clientv2types.NewConfig(s.chainA.SenderAccount.GetAddress().String(), creator.String()))
				_, err := s.chainB.App.GetIBCKeeper().UpdateClientConfig(s.chainB.GetContext(), msg)
				s.Require().NoError(err)
			},
			expError:      nil,
			expAckWritten: true,
		},
		{
			name:     "failure: relayer not permissioned",
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			malleate: func() {
				creator := s.chainB.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointB.ClientID, creator.String(), clientv2types.NewConfig(s.chainA.SenderAccount.GetAddress().String()))
				_, err := s.chainB.App.GetIBCKeeper().UpdateClientConfig(s.chainB.GetContext(), msg)
				s.Require().NoError(err)
			},
			expError: ibcerrors.ErrUnauthorized,
		},
		{
			name:     "failure: counterparty not found",
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			malleate: func() {
				// change the destination id to a non-existent channel.
				packet.DestinationClient = ibctesting.InvalidID
			},
			expError: clientv2types.ErrCounterpartyNotFound,
		},
		{
			name:     "failure: invalid proof",
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			malleate: func() {
				// proof verification fails because the packet commitment is different due to a different sequence.
				packet.Sequence = 10
			},
			expError: commitmenttypes.ErrInvalidProof,
		},
		{
			name:     "failure: invalid acknowledgement",
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			malleate: func() {
				// modify the callback to return the expected recv result.
				path.EndpointB.Chain.GetSimApp().MockModuleV2B.IBCApp.OnRecvPacket = func(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, data types.Payload, relayer sdk.AccAddress) types.RecvPacketResult {
					return types.RecvPacketResult{
						Status:          types.PacketStatus_Success,
						Acknowledgement: []byte(""),
					}
				}
			},
			expError: types.ErrInvalidAcknowledgement,
		},
		{
			name: "failure: async payload with other payloads",
			payloads: []types.Payload{
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
				mockv2.NewAsyncMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
			},
			malleate:      func() {},
			expError:      types.ErrInvalidPacket,
			expAckWritten: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupV2()

			timeoutTimestamp := s.chainA.GetTimeoutTimestampSecs()

			var err error
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, tc.payloads...)
			s.Require().NoError(err)

			// default expected acknowledgement is a single successful acknowledgement for moduleB.
			expAck = types.Acknowledgement{
				AppAcknowledgements: [][]byte{mockv2.MockRecvPacketResult.Acknowledgement},
			}

			tc.malleate()

			// err is checking under expPass
			err = path.EndpointB.MsgRecvPacket(packet)
			ck := path.EndpointB.Chain.GetSimApp().IBCKeeper.ChannelKeeperV2

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)

				// packet receipt should be written
				_, ok := ck.GetPacketReceipt(path.EndpointB.Chain.GetContext(), packet.DestinationClient, packet.Sequence)
				s.Require().True(ok)

				ackWritten := ck.HasPacketAcknowledgement(path.EndpointB.Chain.GetContext(), packet.DestinationClient, packet.Sequence)

				if !tc.expAckWritten {
					// ack should not be written for async app or if the packet receipt was already present.
					s.Require().False(ackWritten)
				} else { // successful or failed acknowledgement
					// ack should be written for synchronous app (default mock application behaviour).
					s.Require().True(ackWritten)
					expectedBz := types.CommitAcknowledgement(expAck)

					actualAckBz := ck.GetPacketAcknowledgement(path.EndpointB.Chain.GetContext(), packet.DestinationClient, packet.Sequence)
					s.Require().Equal(expectedBz, actualAckBz)
				}
			} else {
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expError)
				_, ok := ck.GetPacketReceipt(path.EndpointB.Chain.GetContext(), packet.SourceClient, packet.Sequence)
				s.Require().False(ok)
			}
		})
	}
}

func (s *KeeperTestSuite) TestMsgAcknowledgement() {
	var (
		path   *ibctesting.Path
		packet types.Packet
		ack    types.Acknowledgement
	)
	testCases := []struct {
		name     string
		malleate func()
		payloads []types.Payload
		expError error
	}{
		{
			name:     "success",
			malleate: func() {},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
		},
		{
			name: "success: NoOp",
			malleate: func() {
				s.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeletePacketCommitment(s.chainA.GetContext(), packet.SourceClient, packet.Sequence)

				// Modify the callback to return an error.
				// This way, we can verify that the callback is not executed in a No-op case.
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnAcknowledgementPacket = func(sdk.Context, string, string, uint64, types.Payload, []byte, sdk.AccAddress) error {
					return mockv1.MockApplicationCallbackError
				}
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
		},
		{
			name: "success: failed ack result",
			malleate: func() {
				ack.AppAcknowledgements[0] = types.ErrorAcknowledgement[:]
			},
			payloads: []types.Payload{mockv2.NewErrorMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
		},
		{
			name: "success: multiple payloads",
			malleate: func() {
				ack = types.Acknowledgement{
					AppAcknowledgements: [][]byte{
						mockv2.MockRecvPacketResult.Acknowledgement,
						mockv2.MockRecvPacketResult.Acknowledgement,
					},
				}
			},
			payloads: []types.Payload{
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
			},
		},
		{
			name: "success: multiple payloads with error ack",
			malleate: func() {
				ack = types.Acknowledgement{
					AppAcknowledgements: [][]byte{
						types.ErrorAcknowledgement[:],
					},
				}
			},
			payloads: []types.Payload{
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
				mockv2.NewErrorMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
			},
		},
		{
			name: "success: relayer permissioned with msg sender",
			malleate: func() {
				creator := s.chainA.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, creator.String(), clientv2types.NewConfig(s.chainB.SenderAccount.GetAddress().String(), creator.String()))
				_, err := s.chainA.App.GetIBCKeeper().UpdateClientConfig(s.chainA.GetContext(), msg)
				s.Require().NoError(err)
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
		},
		{
			name: "failure: relayer not permissioned",
			malleate: func() {
				creator := s.chainA.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, creator.String(), clientv2types.NewConfig(s.chainB.SenderAccount.GetAddress().String()))
				_, err := s.chainA.App.GetIBCKeeper().UpdateClientConfig(s.chainA.GetContext(), msg)
				s.Require().NoError(err)
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			expError: ibcerrors.ErrUnauthorized,
		},
		{
			name: "failure: callback fails",
			malleate: func() {
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnAcknowledgementPacket = func(sdk.Context, string, string, uint64, types.Payload, []byte, sdk.AccAddress) error {
					return mockv1.MockApplicationCallbackError
				}
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			expError: mockv1.MockApplicationCallbackError,
		},
		{
			name: "failure: callback fails on one of the multiple payloads",
			malleate: func() {
				// create custom callback that fails on one of the payloads in the test case.
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnAcknowledgementPacket = func(ctx sdk.Context, sourceClient string, destinationClient string, sequence uint64, data types.Payload, acknowledgement []byte, relayer sdk.AccAddress) error {
					if data.DestinationPort == mockv2.ModuleNameB {
						return mockv1.MockApplicationCallbackError
					}
					return nil
				}
				ack = types.Acknowledgement{
					AppAcknowledgements: [][]byte{
						mockv2.MockRecvPacketResult.Acknowledgement,
						mockv2.MockRecvPacketResult.Acknowledgement,
						mockv2.MockRecvPacketResult.Acknowledgement, // this one will not be processed
					},
				}
			},
			payloads: []types.Payload{
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameA),
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameA),
			},
			expError: mockv1.MockApplicationCallbackError,
		},
		{
			name: "failure: counterparty not found",
			malleate: func() {
				// change the source id to a non-existent channel.
				packet.SourceClient = "not-existent-channel"
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			expError: clientv2types.ErrCounterpartyNotFound,
		},
		{
			name: "failure: invalid commitment",
			malleate: func() {
				s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(s.chainA.GetContext(), packet.SourceClient, packet.Sequence, []byte("foo"))
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			expError: types.ErrInvalidPacket,
		},
		{
			name: "failure: failed membership verification",
			malleate: func() {
				ack.AppAcknowledgements[0] = mockv1.MockFailPacketData
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			expError: errors.New("failed packet acknowledgement verification"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupV2()

			timeoutTimestamp := s.chainA.GetTimeoutTimestampSecs()

			var err error
			// Send packet from A to B
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, tc.payloads...)
			s.Require().NoError(err)

			err = path.EndpointB.MsgRecvPacket(packet)
			s.Require().NoError(err)

			// Construct expected acknowledgement
			ack = types.Acknowledgement{AppAcknowledgements: [][]byte{mockv2.MockRecvPacketResult.Acknowledgement}}

			tc.malleate()

			// Finally, acknowledge the packet on A
			err = path.EndpointA.MsgAcknowledgePacket(packet, ack)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
			} else {
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expError, "expected error %q, got %q instead", tc.expError, err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestMsgTimeout() {
	var (
		path   *ibctesting.Path
		packet types.Packet
	)

	testCases := []struct {
		name     string
		malleate func()
		payloads []types.Payload
		expError error
	}{
		{
			name: "success",
			malleate: func() {
				s.Require().NoError(path.EndpointA.UpdateClient())
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
		},
		{
			name: "success: no-op",
			malleate: func() {
				s.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeletePacketCommitment(s.chainA.GetContext(), packet.SourceClient, packet.Sequence)

				// Modify the callback to return a different error.
				// This way, we can verify that the callback is not executed in a No-op case.
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnTimeoutPacket = func(sdk.Context, string, string, uint64, types.Payload, sdk.AccAddress) error {
					return mockv1.MockApplicationCallbackError
				}
				s.Require().NoError(path.EndpointA.UpdateClient())
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
		},
		{
			name: "success: multiple payloads",
			malleate: func() {
				s.Require().NoError(path.EndpointA.UpdateClient())
			},
			payloads: []types.Payload{
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
			},
			expError: nil,
		},
		{
			name: "success: relayer permissioned with msg sender",
			malleate: func() {
				creator := s.chainA.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, creator.String(), clientv2types.NewConfig(s.chainB.SenderAccount.GetAddress().String(), creator.String()))
				_, err := s.chainA.App.GetIBCKeeper().UpdateClientConfig(s.chainA.GetContext(), msg)
				s.Require().NoError(err)
				s.Require().NoError(path.EndpointA.UpdateClient())
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
		},
		{
			name: "failure: relayer not permissioned",
			malleate: func() {
				// update first before permissioning the relayer in this case
				s.Require().NoError(path.EndpointA.UpdateClient())
				creator := s.chainA.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, creator.String(), clientv2types.NewConfig(s.chainB.SenderAccount.GetAddress().String()))
				_, err := s.chainA.App.GetIBCKeeper().UpdateClientConfig(s.chainA.GetContext(), msg)
				s.Require().NoError(err)
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			expError: ibcerrors.ErrUnauthorized,
		},
		{
			name: "failure: callback fails",
			malleate: func() {
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnTimeoutPacket = func(sdk.Context, string, string, uint64, types.Payload, sdk.AccAddress) error {
					return mockv1.MockApplicationCallbackError
				}
				s.Require().NoError(path.EndpointA.UpdateClient())
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			expError: mockv1.MockApplicationCallbackError,
		},
		{
			name: "failure: callback fails on one of the multiple payloads",
			malleate: func() {
				// create custom callback that fails on one of the payloads in the test case.
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnTimeoutPacket = func(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, data types.Payload, relayer sdk.AccAddress) error {
					if bytes.Equal(mockv1.MockFailPacketData, data.Value) {
						return mockv1.MockApplicationCallbackError
					}
					return nil
				}
				s.Require().NoError(path.EndpointA.UpdateClient())
			},
			payloads: []types.Payload{
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
				mockv2.NewErrorMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
				mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB),
			},
			expError: mockv1.MockApplicationCallbackError,
		},
		{
			name: "failure: client not found",
			malleate: func() {
				// change the source id to a non-existent client.
				packet.SourceClient = "not-existent-client"
				s.Require().NoError(path.EndpointA.UpdateClient())
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			expError: clientv2types.ErrCounterpartyNotFound,
		},
		{
			name: "failure: invalid commitment",
			malleate: func() {
				s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(s.chainA.GetContext(), packet.SourceClient, packet.Sequence, []byte("foo"))
				s.Require().NoError(path.EndpointA.UpdateClient())
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			expError: types.ErrInvalidPacket,
		},
		{
			name: "failure: unable to timeout if packet has been received",
			malleate: func() {
				s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
				s.Require().NoError(path.EndpointB.UpdateClient())
				s.Require().NoError(path.EndpointA.UpdateClient())
			},
			payloads: []types.Payload{mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)},
			expError: commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupV2()

			// Send packet from A to B
			// make timeoutTimestamp 1 second more than sending chain time to ensure it passes SendPacket
			// and times out successfully after update
			timeoutTimestamp := uint64(s.chainA.GetContext().BlockTime().Add(time.Second).Unix())

			var err error
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, tc.payloads...)
			s.Require().NoError(err)
			s.Require().NotEmpty(packet)

			tc.malleate()

			err = path.EndpointA.MsgTimeoutPacket(packet)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
			} else {
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expError, "expected error %q, got %q instead", tc.expError, err)
			}
		})
	}
}
