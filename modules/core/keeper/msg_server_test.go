package keeper_test

import (
	"errors"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	internalerrors "github.com/cosmos/ibc-go/v10/modules/core/internal/errors"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
)

var (
	timeoutHeight = clienttypes.NewHeight(1, 10000)
	maxSequence   = uint64(10)
)

// TestRegisterCounterparty tests that counterpartyInfo is correctly stored
// and only if the submittor is the same submittor as prior createClient msg
func (s *KeeperTestSuite) TestRegisterCounterparty() {
	var path *ibctesting.Path
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				path.SetupClients()
			},
			nil,
		},
		{
			"client not created first",
			func() {},
			ibcerrors.ErrUnauthorized,
		},
		{
			"creator is different than expected",
			func() {
				path.SetupClients()
				path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.SetClientCreator(s.chainA.GetContext(), path.EndpointA.ClientID, sdk.AccAddress(ibctesting.TestAccAddress))
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"counterparty already registered",
			func() {
				path.SetupV2()
			},
			ibcerrors.ErrInvalidRequest,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()
			merklePrefix := [][]byte{[]byte("ibc"), []byte("channel-7")}
			msg := clientv2types.NewMsgRegisterCounterparty(path.EndpointA.ClientID, merklePrefix, path.EndpointB.ClientID, s.chainA.SenderAccount.GetAddress().String())
			_, err := s.chainA.App.GetIBCKeeper().RegisterCounterparty(s.chainA.GetContext(), msg)
			if tc.expError != nil {
				s.Require().Error(err)
				s.Require().True(errors.Is(err, tc.expError))
			} else {
				s.Require().NoError(err)
				counterpartyInfo, ok := s.chainA.App.GetIBCKeeper().ClientV2Keeper.GetClientCounterparty(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(ok)
				s.Require().Equal(counterpartyInfo, clientv2types.NewCounterpartyInfo(merklePrefix, path.EndpointB.ClientID))
				nextSeqSend, ok := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetNextSequenceSend(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(ok)
				s.Require().Equal(nextSeqSend, uint64(1))
			}
		})
	}
}

// tests the IBC handler receiving a packet on ordered and unordered channels.
// It verifies that the storing of an acknowledgement on success occurs. It
// tests high level properties like ordering and basic sanity checks. More
// rigorous testing of 'RecvPacket' can be found in the
// 04-channel/keeper/packet_test.go.
func (s *KeeperTestSuite) TestHandleRecvPacket() {
	var (
		packet channeltypes.Packet
		path   *ibctesting.Path
	)

	testCases := []struct {
		name      string
		malleate  func()
		expError  error
		expRevert bool
		async     bool // indicate no ack written
		replay    bool // indicate replay (no-op)
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, false, false, false},
		{"success: UNORDERED", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, false, false, false},
		{"success: UNORDERED out of order packet", func() {
			// setup uses an UNORDERED channel
			path.Setup()

			// attempts to receive packet with sequence 10 without receiving packet with sequence 1
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}
		}, nil, false, false, false},
		{"success: OnRecvPacket callback returns revert=true", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockFailPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockFailPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, true, false, false},
		{"success: ORDERED - async acknowledgement", func() {
			path.SetChannelOrdered()
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibcmock.MockAsyncPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibcmock.MockAsyncPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, false, true, false},
		{"success: UNORDERED - async acknowledgement", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibcmock.MockAsyncPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibcmock.MockAsyncPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, false, true, false},
		{"failure: ORDERED out of order packet", func() {
			path.SetChannelOrdered()
			path.Setup()

			// attempts to receive packet with sequence 10 without receiving packet with sequence 1
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}
		}, errors.New("packet sequence is out of order"), false, false, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			s.Require().NotNil(packet)
		}, errors.New("channel not found"), false, false, false},
		{"packet not sent", func() {
			path.Setup()
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, errors.New("receive packet verification failed: couldn't verify counterparty packet commitment"), false, false, false},
		{"successful no-op: ORDERED - packet already received (replay)", func() {
			// mock will panic if application callback is called twice on the same packet
			path.SetChannelOrdered()
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)
		}, nil, false, false, true},
		{"successful no-op: UNORDERED - packet already received (replay)", func() {
			// mock will panic if application callback is called twice on the same packet
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)
		}, nil, false, false, true},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			var (
				proof       []byte
				proofHeight clienttypes.Height
			)

			// get proof of packet commitment from chainA
			packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			if path.EndpointA.ChannelID != "" {
				proof, proofHeight = path.EndpointA.QueryProof(packetKey)
			}

			msg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, s.chainB.SenderAccount.GetAddress().String())

			ctx := s.chainB.GetContext()
			_, err := s.chainB.App.GetIBCKeeper().RecvPacket(ctx, msg)

			events := ctx.EventManager().Events()

			if tc.expError == nil {
				s.Require().NoError(err)

				// replay should not fail since it will be treated as a no-op
				_, err := s.chainB.App.GetIBCKeeper().RecvPacket(s.chainB.GetContext(), msg)
				s.Require().NoError(err)

				if tc.expRevert {
					// context events should contain error events
					s.Require().Contains(events, internalerrors.ConvertToErrorEvents(sdk.Events{ibcmock.NewMockRecvPacketEvent()})[0])
					s.Require().NotContains(events, ibcmock.NewMockRecvPacketEvent())
				} else {
					if tc.replay {
						// context should not contain application events
						s.Require().NotContains(events, ibcmock.NewMockRecvPacketEvent())
						s.Require().NotContains(events, internalerrors.ConvertToErrorEvents(sdk.Events{ibcmock.NewMockRecvPacketEvent()})[0])
					} else {
						// context events should contain application events
						s.Require().Contains(events, ibcmock.NewMockRecvPacketEvent())
					}
				}

				// verify if ack was written
				ack, found := s.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(s.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

				if tc.async {
					s.Require().Nil(ack)
					s.Require().False(found)
				} else {
					s.Require().NotNil(ack)
					s.Require().True(found)
				}
			} else {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (s *KeeperTestSuite) TestUpdateClient() {
	var path *ibctesting.Path
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: update client, no params",
			func() {},
			nil,
		},
		{
			"success: update client, with v2 params set to correct relayer",
			func() {
				creator := s.chainA.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, creator.String(), clientv2types.NewConfig(s.chainB.SenderAccount.GetAddress().String(), creator.String()))
				_, err := s.chainA.App.GetIBCKeeper().UpdateClientConfig(s.chainA.GetContext(), msg)
				s.Require().NoError(err)
			},
			nil,
		},
		{
			"failure: update client with invalid relayer",
			func() {
				creator := s.chainA.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, creator.String(), clientv2types.NewConfig(s.chainB.SenderAccount.GetAddress().String()))
				_, err := s.chainA.App.GetIBCKeeper().UpdateClientConfig(s.chainA.GetContext(), msg)
				s.Require().NoError(err)
			},
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			tc.malleate()

			err := path.EndpointA.UpdateClient()

			if tc.expError == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (s *KeeperTestSuite) TestRecoverClient() {
	var msg *clienttypes.MsgRecoverClient

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: recover client",
			func() {},
			nil,
		},
		{
			"signer doesn't match authority",
			func() {
				msg.Signer = ibctesting.InvalidID
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"invalid subject client",
			func() {
				msg.SubjectClientId = ibctesting.InvalidID
			},
			clienttypes.ErrRouteNotFound,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			subjectPath := ibctesting.NewPath(s.chainA, s.chainB)
			subjectPath.SetupClients()
			subject := subjectPath.EndpointA.ClientID
			subjectClientState := s.chainA.GetClientState(subject)

			substitutePath := ibctesting.NewPath(s.chainA, s.chainB)
			substitutePath.SetupClients()
			substitute := substitutePath.EndpointA.ClientID

			// update substitute twice
			err := substitutePath.EndpointA.UpdateClient()
			s.Require().NoError(err)
			err = substitutePath.EndpointA.UpdateClient()
			s.Require().NoError(err)

			tmClientState, ok := subjectClientState.(*ibctm.ClientState)
			s.Require().True(ok)
			tmClientState.FrozenHeight = tmClientState.LatestHeight
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), subject, tmClientState)

			msg = clienttypes.NewMsgRecoverClient(s.chainA.App.GetIBCKeeper().GetAuthority(), subject, substitute)

			tc.malleate()

			_, err = s.chainA.App.GetIBCKeeper().RecoverClient(s.chainA.GetContext(), msg)

			if tc.expErr == nil {
				s.Require().NoError(err)

				// Assert that client status is now Active

				lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), subjectPath.EndpointA.ClientID)
				s.Require().NoError(err)
				s.Require().Equal(lightClientModule.Status(s.chainA.GetContext(), subjectPath.EndpointA.ClientID), exported.Active)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// tests the IBC handler acknowledgement of a packet on ordered and unordered
// channels. It verifies that the deletion of packet commitments from state
// occurs. It test high level properties like ordering and basic sanity
// checks. More rigorous testing of 'AcknowledgePacket'
// can be found in the 04-channel/keeper/packet_test.go.
func (s *KeeperTestSuite) TestHandleAcknowledgePacket() {
	var (
		packet channeltypes.Packet
		path   *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
		replay   bool // indicate replay (no-op)
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)
		}, nil, false},
		{"success: UNORDERED", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)
		}, nil, false},
		{"success: UNORDERED acknowledge out of order packet", func() {
			// setup uses an UNORDERED channel
			path.Setup()

			// attempts to acknowledge ack with sequence 10 without acknowledging ack with sequence 1 (removing packet commitment)
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)
			}
		}, nil, false},
		{"failure: ORDERED acknowledge out of order packet", func() {
			path.SetChannelOrdered()
			path.Setup()

			// attempts to acknowledge ack with sequence 10 without acknowledging ack with sequence 1 (removing packet commitment
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)
			}
		}, errors.New("packet sequence is out of order"), false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			s.Require().NotNil(packet)
		}, errors.New("channel not found"), false},
		{"packet not received", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, errors.New("invalid proof"), false},
		{"successful no-op: ORDERED - packet already acknowledged (replay)", func() {
			path.SetChannelOrdered()
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
			s.Require().NoError(err)
		}, nil, true},
		{"successful no-op: UNORDERED - packet already acknowledged (replay)", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
			s.Require().NoError(err)
		}, nil, true},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			var (
				proof       []byte
				proofHeight clienttypes.Height
			)
			packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			if path.EndpointB.ChannelID != "" {
				proof, proofHeight = path.EndpointB.QueryProof(packetKey)
			}

			msg := channeltypes.NewMsgAcknowledgement(packet, ibcmock.MockAcknowledgement.Acknowledgement(), proof, proofHeight, s.chainA.SenderAccount.GetAddress().String())

			ctx := s.chainA.GetContext()
			_, err := s.chainA.App.GetIBCKeeper().Acknowledgement(ctx, msg)

			events := ctx.EventManager().Events()

			if tc.expError == nil {
				s.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := s.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(s.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				s.Require().False(has)

				// replay should not error as it is treated as a no-op
				_, err := s.chainA.App.GetIBCKeeper().Acknowledgement(s.chainA.GetContext(), msg)
				s.Require().NoError(err)

				if tc.replay {
					// context should not contain application events
					s.Require().NotContains(events, ibcmock.NewMockAckPacketEvent())
				} else {
					// context events should contain application events
					s.Require().Contains(events, ibcmock.NewMockAckPacketEvent())
				}
			} else {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

// tests the IBC handler timing out a packet on ordered and unordered channels.
// It verifies that the deletion of a packet commitment occurs. It tests
// high level properties like ordering and basic sanity checks. More
// rigorous testing of 'TimeoutPacket' and 'TimeoutExecuted' can be found in
// the 04-channel/keeper/timeout_test.go.
func (s *KeeperTestSuite) TestHandleTimeoutPacket() {
	var (
		packet    channeltypes.Packet
		packetKey []byte
		path      *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
		noop     bool // indicate no-op
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, nil, false},
		{"success: UNORDERED", func() {
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, nil, false},
		{"success: UNORDERED timeout out of order packet", func() {
			// setup uses an UNORDERED channel
			path.Setup()

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, nil, false},
		{"success: ORDERED timeout out of order packet", func() {
			path.SetChannelOrdered()
			path.Setup()

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, nil, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			s.Require().NotNil(packet)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, errors.New("channel not found"), false},
		{"successful no-op: UNORDERED - packet not sent", func() {
			path.Setup()
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 1), 0)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, nil, true},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			var (
				proof       []byte
				proofHeight clienttypes.Height
			)
			if path.EndpointB.ChannelID != "" {
				proof, proofHeight = path.EndpointB.QueryProof(packetKey)
			}

			msg := channeltypes.NewMsgTimeout(packet, 1, proof, proofHeight, s.chainA.SenderAccount.GetAddress().String())

			ctx := s.chainA.GetContext()
			_, err := s.chainA.App.GetIBCKeeper().Timeout(ctx, msg)

			events := ctx.EventManager().Events()

			if tc.expErr == nil {
				s.Require().NoError(err)

				// replay should not return an error as it is treated as a no-op
				_, err := s.chainA.App.GetIBCKeeper().Timeout(s.chainA.GetContext(), msg)
				s.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := s.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(s.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				s.Require().False(has)

				if tc.noop {
					// context should not contain application events
					s.Require().NotContains(events, ibcmock.NewMockTimeoutPacketEvent())
				} else {
					// context should contain application events
					s.Require().Contains(events, ibcmock.NewMockTimeoutPacketEvent())
				}
			} else {
				s.Require().Error(err)

				s.Require().Contains(err.Error(), tc.expErr.Error())
			}
		})
	}
}

// tests the IBC handler timing out a packet via channel closure on ordered
// and unordered channels. It verifies that the deletion of a packet
// commitment occurs. It tests high level properties like ordering and basic
// sanity checks. More rigorous testing of 'TimeoutOnClose' and
// 'TimeoutExecuted' can be found in the 04-channel/keeper/timeout_test.go.
func (s *KeeperTestSuite) TestHandleTimeoutOnClosePacket() {
	var (
		packet    channeltypes.Packet
		packetKey []byte
		path      *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			path.Setup()

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())

			// close counterparty channel
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		}, nil},
		{"success: UNORDERED", func() {
			path.Setup()

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		}, nil},
		{"success: UNORDERED timeout out of order packet", func() {
			// setup uses an UNORDERED channel
			path.Setup()

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		}, nil},
		{"success: ORDERED timeout out of order packet", func() {
			path.SetChannelOrdered()
			path.Setup()

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())

			// close counterparty channel
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		}, nil},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			s.Require().NotNil(packet)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, errors.New("channel not found")},
		{"successful no-op: UNORDERED - packet not sent", func() {
			path.Setup()
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 1), 0)
			packetKey = host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		}, nil},
		{"ORDERED: channel not closed", func() {
			path.SetChannelOrdered()
			path.Setup()

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, errors.New("invalid proof")},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			proof, proofHeight := s.chainB.QueryProof(packetKey)

			channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			closedProof, _ := s.chainB.QueryProof(channelKey)

			msg := channeltypes.NewMsgTimeoutOnClose(packet, 1, proof, closedProof, proofHeight, s.chainA.SenderAccount.GetAddress().String())

			_, err := s.chainA.App.GetIBCKeeper().TimeoutOnClose(s.chainA.GetContext(), msg)

			if tc.expError == nil {
				s.Require().NoError(err)

				// replay should not return an error as it will be treated as a no-op
				_, err := s.chainA.App.GetIBCKeeper().TimeoutOnClose(s.chainA.GetContext(), msg)
				s.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := s.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(s.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				s.Require().False(has)
			} else {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (s *KeeperTestSuite) TestUpgradeClient() {
	var (
		path              *ibctesting.Path
		newChainID        string
		newClientHeight   clienttypes.Height
		upgradedClient    *ibctm.ClientState
		upgradedConsState exported.ConsensusState
		lastHeight        exported.Height
		msg               *clienttypes.MsgUpgradeClient
	)
	cases := []struct {
		name   string
		setup  func()
		expErr error
	}{
		{
			name: "successful upgrade",
			setup: func() {
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod+ibctesting.TrustingPeriod, ibctesting.MaxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				// Call ZeroCustomFields on upgraded clients to clear any client-chosen parameters in test-case upgradedClient
				upgradedClient = upgradedClient.ZeroCustomFields()

				upgradedConsState = &ibctm.ConsensusState{
					NextValidatorsHash: []byte("nextValsHash"),
				}

				// last Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				upgradedClientBz, err := clienttypes.MarshalClientState(s.chainA.App.AppCodec(), upgradedClient)
				s.Require().NoError(err)
				upgradedConsStateBz, err := clienttypes.MarshalConsensusState(s.chainA.App.AppCodec(), upgradedConsState)
				s.Require().NoError(err)

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for testing
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for testing

				// commit upgrade store changes and update clients
				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				latestHeight := path.EndpointA.GetClientLatestHeight()
				upgradeClientProof, _ := s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), latestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ := s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), latestHeight.GetRevisionHeight())

				msg, err = clienttypes.NewMsgUpgradeClient(path.EndpointA.ClientID, upgradedClient, upgradedConsState,
					upgradeClientProof, upgradedConsensusStateProof, s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			expErr: nil,
		},
		{
			name: "VerifyUpgrade fails",
			setup: func() {
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod+ibctesting.TrustingPeriod, ibctesting.MaxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				// Call ZeroCustomFields on upgraded clients to clear any client-chosen parameters in test-case upgradedClient
				upgradedClient = upgradedClient.ZeroCustomFields()

				upgradedConsState = &ibctm.ConsensusState{
					NextValidatorsHash: []byte("nextValsHash"),
				}

				// last Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				upgradedClientBz, err := clienttypes.MarshalClientState(s.chainA.App.AppCodec(), upgradedClient)
				s.Require().NoError(err)
				upgradedConsStateBz, err := clienttypes.MarshalConsensusState(s.chainA.App.AppCodec(), upgradedConsState)
				s.Require().NoError(err)

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for testing
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for testing

				// commit upgrade store changes and update clients
				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				msg, err = clienttypes.NewMsgUpgradeClient(path.EndpointA.ClientID, upgradedClient, upgradedConsState, nil, nil, s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			expErr: errors.New("invalid merkle proof"),
		},
	}

	for _, tc := range cases {
		path = ibctesting.NewPath(s.chainA, s.chainB)
		path.SetupClients()

		var err error
		clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
		s.Require().True(ok)
		revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

		newChainID, err = clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
		s.Require().NoError(err)

		newClientHeight = clienttypes.NewHeight(revisionNumber+1, clientState.LatestHeight.GetRevisionHeight()+1)

		tc.setup()

		ctx := s.chainA.GetContext()
		_, err = s.chainA.GetSimApp().GetIBCKeeper().UpgradeClient(ctx, msg)

		if tc.expErr == nil {
			s.Require().NoError(err, "upgrade handler failed on valid case: %s", tc.name)
			newClient, ok := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().True(ok)
			newChainSpecifiedClient := newClient.(*ibctm.ClientState).ZeroCustomFields()
			s.Require().Equal(upgradedClient, newChainSpecifiedClient)

			expectedEvents := sdk.Events{
				sdk.NewEvent(
					clienttypes.EventTypeUpgradeClient,
					sdk.NewAttribute(clienttypes.AttributeKeyClientID, ibctesting.FirstClientID),
					sdk.NewAttribute(clienttypes.AttributeKeyClientType, path.EndpointA.GetClientState().ClientType()),
					sdk.NewAttribute(clienttypes.AttributeKeyConsensusHeight, path.EndpointA.GetClientLatestHeight().String()),
				),
			}.ToABCIEvents()

			expectedEvents = sdk.MarkEventsToIndex(expectedEvents, map[string]struct{}{})
			ibctesting.AssertEvents(&s.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())
		} else {
			s.Require().Error(err, "upgrade handler passed on invalid case: %s", tc.name)
			s.Require().Contains(err.Error(), tc.expErr.Error())
		}
	}
}

// TestIBCSoftwareUpgrade tests the IBCSoftwareUpgrade rpc handler
func (s *KeeperTestSuite) TestIBCSoftwareUpgrade() {
	var msg *clienttypes.MsgIBCSoftwareUpgrade
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: valid authority and client upgrade",
			func() {},
			nil,
		},
		{
			"failure: invalid authority address",
			func() {
				msg.Signer = s.chainA.SenderAccount.GetAddress().String()
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: invalid clientState",
			func() {
				msg.UpgradedClientState = nil
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"failure: failed to schedule client upgrade",
			func() {
				msg.Plan.Height = 0
			},
			sdkerrors.ErrInvalidRequest,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			path := ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()
			validAuthority := s.chainA.App.GetIBCKeeper().GetAuthority()
			plan := upgradetypes.Plan{
				Name:   "upgrade IBC clients",
				Height: 1000,
			}
			// update trusting period
			clientState, ok := path.EndpointB.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			clientState.TrustingPeriod += 100

			var err error
			msg, err = clienttypes.NewMsgIBCSoftwareUpgrade(
				validAuthority,
				plan,
				clientState,
			)

			s.Require().NoError(err)

			tc.malleate()

			_, err = s.chainA.App.GetIBCKeeper().IBCSoftwareUpgrade(s.chainA.GetContext(), msg)

			if tc.expError == nil {
				s.Require().NoError(err)
				// upgrade plan is stored
				storedPlan, err := s.chainA.GetSimApp().UpgradeKeeper.GetUpgradePlan(s.chainA.GetContext())
				s.Require().NoError(err)
				s.Require().Equal(plan, storedPlan)

				// upgraded client state is stored
				bz, err := s.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(s.chainA.GetContext(), plan.Height)
				s.Require().NoError(err)
				upgradedClientState, err := clienttypes.UnmarshalClientState(s.chainA.App.AppCodec(), bz)
				s.Require().NoError(err)
				s.Require().Equal(clientState.ZeroCustomFields(), upgradedClientState)
			} else {
				s.Require().True(errors.Is(err, tc.expError))
			}
		})
	}
}

// TestUpdateClientParams tests the UpdateClientParams rpc handler
func (s *KeeperTestSuite) TestUpdateClientParams() {
	signer := s.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name     string
		msg      *clienttypes.MsgUpdateParams
		expError error
	}{
		{
			"success: valid signer and default params",
			clienttypes.NewMsgUpdateParams(signer, clienttypes.DefaultParams()),
			nil,
		},
		{
			"failure: malformed signer address",
			clienttypes.NewMsgUpdateParams(ibctesting.InvalidID, clienttypes.DefaultParams()),
			errors.New("unauthorized"),
		},
		{
			"failure: empty signer address",
			clienttypes.NewMsgUpdateParams("", clienttypes.DefaultParams()),
			errors.New("unauthorized"),
		},
		{
			"failure: whitespace signer address",
			clienttypes.NewMsgUpdateParams("    ", clienttypes.DefaultParams()),
			errors.New("unauthorized"),
		},
		{
			"failure: unauthorized signer address",
			clienttypes.NewMsgUpdateParams(ibctesting.TestAccAddress, clienttypes.DefaultParams()),
			errors.New("unauthorized"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			_, err := s.chainA.App.GetIBCKeeper().UpdateClientParams(s.chainA.GetContext(), tc.msg)
			if tc.expError == nil {
				s.Require().NoError(err)
				p := s.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(s.chainA.GetContext())
				s.Require().Equal(tc.msg.Params, p)
			} else {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

// TestUpdateConnectionParams tests the UpdateConnectionParams rpc handler
func (s *KeeperTestSuite) TestUpdateConnectionParams() {
	signer := s.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name   string
		msg    *connectiontypes.MsgUpdateParams
		expErr error
	}{
		{
			"success: valid signer and default params",
			connectiontypes.NewMsgUpdateParams(signer, connectiontypes.DefaultParams()),
			nil,
		},
		{
			"failure: malformed signer address",
			connectiontypes.NewMsgUpdateParams(ibctesting.InvalidID, connectiontypes.DefaultParams()),
			errors.New("unauthorized"),
		},
		{
			"failure: empty signer address",
			connectiontypes.NewMsgUpdateParams("", connectiontypes.DefaultParams()),
			errors.New("unauthorized"),
		},
		{
			"failure: whitespace signer address",
			connectiontypes.NewMsgUpdateParams("    ", connectiontypes.DefaultParams()),
			errors.New("unauthorized"),
		},
		{
			"failure: unauthorized signer address",
			connectiontypes.NewMsgUpdateParams(ibctesting.TestAccAddress, connectiontypes.DefaultParams()),
			errors.New("unauthorized"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			_, err := s.chainA.App.GetIBCKeeper().UpdateConnectionParams(s.chainA.GetContext(), tc.msg)
			if tc.expErr == nil {
				s.Require().NoError(err)
				p := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetParams(s.chainA.GetContext())
				s.Require().Equal(tc.msg.Params, p)
			} else {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (s *KeeperTestSuite) TestUpdateClientConfig() {
	var (
		path   *ibctesting.Path
		signer string
		config clientv2types.Config
	)
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: valid authority and default config",
			func() {
				signer = s.chainA.App.GetIBCKeeper().GetAuthority()
			},
			nil,
		},
		{
			"success: valid creator and default config",
			func() {
				signer = s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientCreator(s.chainA.GetContext(), path.EndpointA.ClientID).String()
			},
			nil,
		},
		{
			"success: valid authority and custom config",
			func() {
				signer = s.chainA.App.GetIBCKeeper().GetAuthority()
				config = clientv2types.NewConfig(s.chainB.SenderAccount.String(), s.chainA.SenderAccount.String())
			},
			nil,
		},
		{
			"success: valid creator and default config",
			func() {
				signer = s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientCreator(s.chainA.GetContext(), path.EndpointA.ClientID).String()
				config = clientv2types.NewConfig(s.chainB.SenderAccount.String(), s.chainA.SenderAccount.String())
			},
			nil,
		},
		{
			"success: valid creator and setting config to empty after it has been set",
			func() {
				signer = s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientCreator(s.chainA.GetContext(), path.EndpointA.ClientID).String()
				config = clientv2types.NewConfig(s.chainB.SenderAccount.String(), s.chainA.SenderAccount.String())
				_, err := s.chainA.App.GetIBCKeeper().UpdateClientConfig(s.chainA.GetContext(), clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, signer, config))
				s.Require().NoError(err)
				config = clientv2types.DefaultConfig()
			},
			nil,
		},
		{
			"success: valid creator and setting config to different config after it has been set",
			func() {
				signer = s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientCreator(s.chainA.GetContext(), path.EndpointA.ClientID).String()
				config = clientv2types.NewConfig(s.chainA.SenderAccount.String())
				_, err := s.chainA.App.GetIBCKeeper().UpdateClientConfig(s.chainA.GetContext(), clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, signer, config))
				s.Require().NoError(err)
				config = clientv2types.NewConfig(s.chainB.SenderAccount.String(), s.chainA.SenderAccount.String())
			},
			nil,
		},
		{
			"failure: invalid signer",
			func() {
				signer = s.chainB.SenderAccount.GetAddress().String()
				config = clientv2types.NewConfig(s.chainB.SenderAccount.String())
			},
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			config = clientv2types.DefaultConfig()

			tc.malleate()

			msg := clientv2types.NewMsgUpdateClientConfig(path.EndpointA.ClientID, signer, config)
			_, err := s.chainA.App.GetIBCKeeper().UpdateClientConfig(s.chainA.GetContext(), msg)
			if tc.expError == nil {
				s.Require().NoError(err)
				c := s.chainA.App.GetIBCKeeper().ClientV2Keeper.GetConfig(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().Equal(config, c)
			} else {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

// TestDeleteClientCreator tests the DeleteClientCreator message handler
func (s *KeeperTestSuite) TestDeleteClientCreator() {
	var (
		path     *ibctesting.Path
		clientID string
		msg      *clienttypes.MsgDeleteClientCreator
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: valid creator deletes itself",
			func() {
				creator := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientCreator(s.chainA.GetContext(), clientID)
				msg = clienttypes.NewMsgDeleteClientCreator(clientID, creator.String())
			},
			nil,
		},
		{
			"success: valid authority deletes client creator",
			func() {
				msg = clienttypes.NewMsgDeleteClientCreator(clientID, s.chainA.App.GetIBCKeeper().GetAuthority())
			},
			nil,
		},
		{
			"failure: deleting a client creator that was already deleted",
			func() {
				// First delete the creator
				authority := s.chainA.App.GetIBCKeeper().GetAuthority()
				deleteMsg := clienttypes.NewMsgDeleteClientCreator(clientID, authority)
				_, err := s.chainA.App.GetIBCKeeper().DeleteClientCreator(s.chainA.GetContext(), deleteMsg)
				s.Require().NoError(err)

				// Now try to delete it again
				msg = clienttypes.NewMsgDeleteClientCreator(clientID, authority)
			},
			ibcerrors.ErrNotFound, // Now it should fail with not found
		},
		{
			"failure: unauthorized signer - not creator or authority",
			func() {
				msg = clienttypes.NewMsgDeleteClientCreator(clientID, s.chainB.SenderAccount.GetAddress().String())
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: client ID does not exist",
			func() {
				msg = clienttypes.NewMsgDeleteClientCreator("nonexistentclient", s.chainA.App.GetIBCKeeper().GetAuthority())
			},
			ibcerrors.ErrNotFound,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()
			clientID = path.EndpointA.ClientID

			tc.malleate()

			_, err := s.chainA.App.GetIBCKeeper().DeleteClientCreator(s.chainA.GetContext(), msg)

			if tc.expError == nil {
				s.Require().NoError(err)

				// Verify creator has been deleted
				creator := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientCreator(s.chainA.GetContext(), clientID)
				s.Require().Nil(creator)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
