package keeper_test

import (
	"context"
	"errors"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	abci "github.com/cometbft/cometbft/api/cometbft/abci/v1"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	ibcmock "github.com/cosmos/ibc-go/v9/testing/mock"
)

var (
	timeoutHeight = clienttypes.NewHeight(1, 10000)
	maxSequence   = uint64(10)
)

// tests the IBC handler receiving a packet on ordered and unordered channels.
// It verifies that the storing of an acknowledgement on success occurs. It
// tests high level properties like ordering and basic sanity checks. More
// rigorous testing of 'RecvPacket' can be found in the
// 04-channel/keeper/packet_test.go.
func (suite *KeeperTestSuite) TestHandleRecvPacket() {
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
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, false, false, false},
		{"success: UNORDERED", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, false, false, false},
		{"success: UNORDERED out of order packet", func() {
			// setup uses an UNORDERED channel
			path.Setup()

			// attempts to receive packet with sequence 10 without receiving packet with sequence 1
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}
		}, nil, false, false, false},
		{"success: OnRecvPacket callback returns revert=true", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockFailPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockFailPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, true, false, false},
		{"success: ORDERED - async acknowledgement", func() {
			path.SetChannelOrdered()
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibcmock.MockAsyncPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibcmock.MockAsyncPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, false, true, false},
		{"success: UNORDERED - async acknowledgement", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibcmock.MockAsyncPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibcmock.MockAsyncPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, false, true, false},
		{"failure: ORDERED out of order packet", func() {
			path.SetChannelOrdered()
			path.Setup()

			// attempts to receive packet with sequence 10 without receiving packet with sequence 1
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}
		}, errors.New("packet sequence is out of order"), false, false, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)
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
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, nil, false, false, true},
		{"successful no-op: UNORDERED - packet already received (replay)", func() {
			// mock will panic if application callback is called twice on the same packet
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, nil, false, false, true},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

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

			msg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, suite.chainB.SenderAccount.GetAddress().String())

			ctx := suite.chainB.GetContext()
			_, err := suite.chainB.App.GetIBCKeeper().RecvPacket(ctx, msg)

			events := ctx.EventManager().Events()

			if tc.expError == nil {
				suite.Require().NoError(err)

				// replay should not fail since it will be treated as a no-op
				_, err := suite.chainB.App.GetIBCKeeper().RecvPacket(suite.chainB.GetContext(), msg)
				suite.Require().NoError(err)

				if tc.expRevert {
					// context events should contain error events
					suite.Require().Contains(events, keeper.ConvertToErrorEvents(sdk.Events{ibcmock.NewMockRecvPacketEvent()})[0])
					suite.Require().NotContains(events, ibcmock.NewMockRecvPacketEvent())
				} else {
					if tc.replay {
						// context should not contain application events
						suite.Require().NotContains(events, ibcmock.NewMockRecvPacketEvent())
						suite.Require().NotContains(events, keeper.ConvertToErrorEvents(sdk.Events{ibcmock.NewMockRecvPacketEvent()})[0])
					} else {
						// context events should contain application events
						suite.Require().Contains(events, ibcmock.NewMockRecvPacketEvent())
					}
				}

				// verify if ack was written
				ack, found := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

				if tc.async {
					suite.Require().Nil(ack)
					suite.Require().False(found)

				} else {
					suite.Require().NotNil(ack)
					suite.Require().True(found)
				}
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestRecoverClient() {
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
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			subjectPath.SetupClients()
			subject := subjectPath.EndpointA.ClientID
			subjectClientState := suite.chainA.GetClientState(subject)

			substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			substitutePath.SetupClients()
			substitute := substitutePath.EndpointA.ClientID

			// update substitute twice
			err := substitutePath.EndpointA.UpdateClient()
			suite.Require().NoError(err)
			err = substitutePath.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			tmClientState, ok := subjectClientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			tmClientState.FrozenHeight = tmClientState.LatestHeight
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subject, tmClientState)

			msg = clienttypes.NewMsgRecoverClient(suite.chainA.App.GetIBCKeeper().GetAuthority(), subject, substitute)

			tc.malleate()

			_, err = suite.chainA.App.GetIBCKeeper().RecoverClient(suite.chainA.GetContext(), msg)

			if tc.expErr == nil {
				suite.Require().NoError(err)

				// Assert that client status is now Active

				lightClientModule, err := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)
				suite.Require().NoError(err)
				suite.Require().Equal(lightClientModule.Status(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID), exported.Active)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// tests the IBC handler acknowledgement of a packet on ordered and unordered
// channels. It verifies that the deletion of packet commitments from state
// occurs. It test high level properties like ordering and basic sanity
// checks. More rigorous testing of 'AcknowledgePacket'
// can be found in the 04-channel/keeper/packet_test.go.
func (suite *KeeperTestSuite) TestHandleAcknowledgePacket() {
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
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, nil, false},
		{"success: UNORDERED", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, nil, false},
		{"success: UNORDERED acknowledge out of order packet", func() {
			// setup uses an UNORDERED channel
			path.Setup()

			// attempts to acknowledge ack with sequence 10 without acknowledging ack with sequence 1 (removing packet commitment)
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)
			}
		}, nil, false},
		{"failure: ORDERED acknowledge out of order packet", func() {
			path.SetChannelOrdered()
			path.Setup()

			// attempts to acknowledge ack with sequence 10 without acknowledging ack with sequence 1 (removing packet commitment
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)
			}
		}, errors.New("packet sequence is out of order"), false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)
		}, errors.New("channel not found"), false},
		{"packet not received", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, errors.New("invalid proof"), false},
		{"successful no-op: ORDERED - packet already acknowledged (replay)", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
			suite.Require().NoError(err)
		}, nil, true},
		{"successful no-op: UNORDERED - packet already acknowledged (replay)", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
			suite.Require().NoError(err)
		}, nil, true},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			var (
				proof       []byte
				proofHeight clienttypes.Height
			)
			packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			if path.EndpointB.ChannelID != "" {
				proof, proofHeight = path.EndpointB.QueryProof(packetKey)
			}

			msg := channeltypes.NewMsgAcknowledgement(packet, ibcmock.MockAcknowledgement.Acknowledgement(), proof, proofHeight, suite.chainA.SenderAccount.GetAddress().String())

			ctx := suite.chainA.GetContext()
			_, err := suite.chainA.App.GetIBCKeeper().Acknowledgement(ctx, msg)

			events := ctx.EventManager().Events()

			if tc.expError == nil {
				suite.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				suite.Require().False(has)

				// replay should not error as it is treated as a no-op
				_, err := suite.chainA.App.GetIBCKeeper().Acknowledgement(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				if tc.replay {
					// context should not contain application events
					suite.Require().NotContains(events, ibcmock.NewMockAckPacketEvent())
				} else {
					// context events should contain application events
					suite.Require().Contains(events, ibcmock.NewMockAckPacketEvent())
				}
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

// tests the IBC handler timing out a packet on ordered and unordered channels.
// It verifies that the deletion of a packet commitment occurs. It tests
// high level properties like ordering and basic sanity checks. More
// rigorous testing of 'TimeoutPacket' and 'TimeoutExecuted' can be found in
// the 04-channel/keeper/timeout_test.go.
func (suite *KeeperTestSuite) TestHandleTimeoutPacket() {
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

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, nil, false},
		{"success: UNORDERED", func() {
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, nil, false},
		{"success: UNORDERED timeout out of order packet", func() {
			// setup uses an UNORDERED channel
			path.Setup()

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, nil, false},
		{"success: ORDERED timeout out of order packet", func() {
			path.SetChannelOrdered()
			path.Setup()

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, nil, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, errors.New("channel not found"), false},
		{"successful no-op: UNORDERED - packet not sent", func() {
			path.Setup()
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 1), 0)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, nil, true},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			var (
				proof       []byte
				proofHeight clienttypes.Height
			)
			if path.EndpointB.ChannelID != "" {
				proof, proofHeight = path.EndpointB.QueryProof(packetKey)
			}

			msg := channeltypes.NewMsgTimeout(packet, 1, proof, proofHeight, suite.chainA.SenderAccount.GetAddress().String())

			ctx := suite.chainA.GetContext()
			_, err := suite.chainA.App.GetIBCKeeper().Timeout(ctx, msg)

			events := ctx.EventManager().Events()

			if tc.expErr == nil {
				suite.Require().NoError(err)

				// replay should not return an error as it is treated as a no-op
				_, err := suite.chainA.App.GetIBCKeeper().Timeout(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				suite.Require().False(has)

				if tc.noop {
					// context should not contain application events
					suite.Require().NotContains(events, ibcmock.NewMockTimeoutPacketEvent())
				} else {
					// context should contain application events
					suite.Require().Contains(events, ibcmock.NewMockTimeoutPacketEvent())
				}

			} else {
				suite.Require().Error(err)

				suite.Require().Contains(err.Error(), tc.expErr.Error())
			}
		})
	}
}

// tests the IBC handler timing out a packet via channel closure on ordered
// and unordered channels. It verifies that the deletion of a packet
// commitment occurs. It tests high level properties like ordering and basic
// sanity checks. More rigorous testing of 'TimeoutOnClose' and
// 'TimeoutExecuted' can be found in the 04-channel/keeper/timeout_test.go.
func (suite *KeeperTestSuite) TestHandleTimeoutOnClosePacket() {
	var (
		packet                      channeltypes.Packet
		packetKey                   []byte
		path                        *ibctesting.Path
		counterpartyUpgradeSequence uint64
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
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())

			// close counterparty channel
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		}, nil},
		{"success: UNORDERED", func() {
			path.Setup()

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

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
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

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
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())

			// close counterparty channel
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		}, nil},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)

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
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, errors.New("invalid proof")},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			proof, proofHeight := suite.chainB.QueryProof(packetKey)

			channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			closedProof, _ := suite.chainB.QueryProof(channelKey)

			msg := channeltypes.NewMsgTimeoutOnClose(packet, 1, proof, closedProof, proofHeight, suite.chainA.SenderAccount.GetAddress().String(), counterpartyUpgradeSequence)

			_, err := suite.chainA.App.GetIBCKeeper().TimeoutOnClose(suite.chainA.GetContext(), msg)

			if tc.expError == nil {
				suite.Require().NoError(err)

				// replay should not return an error as it will be treated as a no-op
				_, err := suite.chainA.App.GetIBCKeeper().TimeoutOnClose(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				suite.Require().False(has)

			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUpgradeClient() {
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
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				upgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
				suite.Require().NoError(err)
				upgradedConsStateBz, err := clienttypes.MarshalConsensusState(suite.chainA.App.AppCodec(), upgradedConsState)
				suite.Require().NoError(err)

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for testing
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for testing

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				latestHeight := path.EndpointA.GetClientLatestHeight()
				upgradeClientProof, _ := suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), latestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ := suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), latestHeight.GetRevisionHeight())

				msg, err = clienttypes.NewMsgUpgradeClient(path.EndpointA.ClientID, upgradedClient, upgradedConsState,
					upgradeClientProof, upgradedConsensusStateProof, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
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
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				upgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
				suite.Require().NoError(err)
				upgradedConsStateBz, err := clienttypes.MarshalConsensusState(suite.chainA.App.AppCodec(), upgradedConsState)
				suite.Require().NoError(err)

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for testing
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for testing

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				msg, err = clienttypes.NewMsgUpgradeClient(path.EndpointA.ClientID, upgradedClient, upgradedConsState, nil, nil, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			expErr: errors.New("invalid merkle proof"),
		},
	}

	for _, tc := range cases {
		tc := tc
		path = ibctesting.NewPath(suite.chainA, suite.chainB)
		path.SetupClients()

		var err error
		clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
		suite.Require().True(ok)
		revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

		newChainID, err = clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
		suite.Require().NoError(err)

		newClientHeight = clienttypes.NewHeight(revisionNumber+1, clientState.LatestHeight.GetRevisionHeight()+1)

		tc.setup()

		ctx := suite.chainA.GetContext()
		_, err = suite.chainA.GetSimApp().GetIBCKeeper().UpgradeClient(ctx, msg)

		if tc.expErr == nil {
			suite.Require().NoError(err, "upgrade handler failed on valid case: %s", tc.name)
			newClient, ok := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
			suite.Require().True(ok)
			newChainSpecifiedClient := newClient.(*ibctm.ClientState).ZeroCustomFields()
			suite.Require().Equal(upgradedClient, newChainSpecifiedClient)

			expectedEvents := sdk.Events{
				sdk.NewEvent(
					clienttypes.EventTypeUpgradeClient,
					sdk.NewAttribute(clienttypes.AttributeKeyClientID, ibctesting.FirstClientID),
					sdk.NewAttribute(clienttypes.AttributeKeyClientType, path.EndpointA.GetClientState().ClientType()),
					sdk.NewAttribute(clienttypes.AttributeKeyConsensusHeight, path.EndpointA.GetClientLatestHeight().String()),
				),
			}.ToABCIEvents()

			expectedEvents = sdk.MarkEventsToIndex(expectedEvents, map[string]struct{}{})
			ibctesting.AssertEvents(&suite.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())
		} else {
			suite.Require().Error(err, "upgrade handler passed on invalid case: %s", tc.name)
			suite.Require().Contains(err.Error(), tc.expErr.Error())
		}
	}
}

func (suite *KeeperTestSuite) TestChannelUpgradeInit() {
	var (
		path *ibctesting.Path
		msg  *channeltypes.MsgChannelUpgradeInit
	)

	cases := []struct {
		name      string
		malleate  func()
		expResult func(res *channeltypes.MsgChannelUpgradeInitResponse, events []abci.Event, err error)
	}{
		{
			"success",
			func() {
				msg = channeltypes.NewMsgChannelUpgradeInit(
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					path.EndpointA.GetProposedUpgrade().Fields,
					path.EndpointA.Chain.GetSimApp().IBCKeeper.GetAuthority(),
				)
			},
			func(res *channeltypes.MsgChannelUpgradeInitResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(uint64(1), res.UpgradeSequence)
				channel := path.EndpointA.GetChannel()

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeInit,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"authority is not signer of the upgrade init msg",
			func() {
				msg = channeltypes.NewMsgChannelUpgradeInit(
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					path.EndpointA.GetProposedUpgrade().Fields,
					path.EndpointA.Chain.SenderAccount.String(),
				)
			},
			func(res *channeltypes.MsgChannelUpgradeInitResponse, events []abci.Event, err error) {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, ibcerrors.ErrUnauthorized.Error())
				suite.Require().Nil(res)

				suite.Require().Empty(events)
			},
		},
		{
			"ibc application does not implement the UpgradeableModule interface",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.EndpointA.ChannelConfig.PortID = ibcmock.MockBlockUpgrade
				path.EndpointB.ChannelConfig.PortID = ibcmock.MockBlockUpgrade

				path.Setup()

				msg = channeltypes.NewMsgChannelUpgradeInit(
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					path.EndpointA.GetProposedUpgrade().Fields,
					path.EndpointA.Chain.GetSimApp().IBCKeeper.GetAuthority(),
				)
			},
			func(res *channeltypes.MsgChannelUpgradeInitResponse, events []abci.Event, err error) {
				suite.Require().ErrorIs(err, porttypes.ErrInvalidRoute)
				suite.Require().Nil(res)

				suite.Require().Empty(events)
			},
		},
		{
			"ibc application does not commit state changes in callback",
			func() {
				msg = channeltypes.NewMsgChannelUpgradeInit(
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					path.EndpointA.GetProposedUpgrade().Fields,
					path.EndpointA.Chain.GetSimApp().IBCKeeper.GetAuthority(),
				)

				suite.chainA.GetSimApp().IBCMockModule.IBCApp.OnChanUpgradeInit = func(ctx context.Context, portID, channelID string, order channeltypes.Order, connectionHops []string, version string) (string, error) {
					store := suite.chainA.GetSimApp().GetIBCKeeper().KVStoreService.OpenKVStore(ctx)
					err := store.Set(ibcmock.TestKey, ibcmock.TestValue)
					suite.Require().NoError(err)

					eventService := suite.chainA.GetSimApp().GetIBCKeeper().EventService
					err = eventService.EventManager(ctx).EmitKV(ibcmock.MockEventType)
					suite.Require().NoError(err)
					return ibcmock.UpgradeVersion, nil
				}
			},
			func(res *channeltypes.MsgChannelUpgradeInitResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(uint64(1), res.UpgradeSequence)

				storeKey := suite.chainA.GetSimApp().GetKey(exported.ModuleName)
				store := suite.chainA.GetContext().KVStore(storeKey)
				suite.Require().Nil(store.Get(ibcmock.TestKey))

				for _, event := range events {
					if event.GetType() == ibcmock.MockEventType {
						suite.Fail("expected application callback events to be discarded")
					}
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			// configure the channel upgrade version on testing endpoints
			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().GetIBCKeeper().ChannelUpgradeInit(ctx, msg)
			events := ctx.EventManager().Events().ToABCIEvents()

			tc.expResult(res, events, err)
		})
	}
}

func (suite *KeeperTestSuite) TestChannelUpgradeTry() {
	var (
		path *ibctesting.Path
		msg  *channeltypes.MsgChannelUpgradeTry
	)

	cases := []struct {
		name      string
		malleate  func()
		expResult func(res *channeltypes.MsgChannelUpgradeTryResponse, events []abci.Event, err error)
	}{
		{
			"success",
			func() {},
			func(res *channeltypes.MsgChannelUpgradeTryResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)

				channel := path.EndpointB.GetChannel()
				suite.Require().Equal(channeltypes.FLUSHING, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeTry,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"unsynchronized upgrade sequence writes upgrade error receipt",
			func() {
				path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.UpgradeSequence = 99 })
			},
			func(res *channeltypes.MsgChannelUpgradeTryResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)

				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.FAILURE, res.Result)

				errorReceipt, found := suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				suite.Require().True(found)
				suite.Require().Equal(uint64(99), errorReceipt.Sequence)

				channel := path.EndpointB.GetChannel()
				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeError,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
						// need to manually insert this because the errorReceipt is a string constant as it is written into state
						sdk.NewAttribute(channeltypes.AttributeKeyErrorReceipt, "counterparty upgrade sequence < current upgrade sequence (1 < 99): invalid upgrade sequence"),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"ibc application does not implement the UpgradeableModule interface",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.EndpointA.ChannelConfig.PortID = ibcmock.MockBlockUpgrade
				path.EndpointB.ChannelConfig.PortID = ibcmock.MockBlockUpgrade

				path.Setup()

				msg.PortId = path.EndpointB.ChannelConfig.PortID
				msg.ChannelId = path.EndpointB.ChannelID
			},
			func(res *channeltypes.MsgChannelUpgradeTryResponse, events []abci.Event, err error) {
				suite.Require().ErrorIs(err, porttypes.ErrInvalidRoute)
				suite.Require().Nil(res)

				suite.Require().Empty(events)
			},
		},
		{
			"ibc application does not commit state changes in callback",
			func() {
				suite.chainA.GetSimApp().IBCMockModule.IBCApp.OnChanUpgradeTry = func(ctx context.Context, portID, channelID string, order channeltypes.Order, connectionHops []string, counterpartyVersion string) (string, error) {
					store := suite.chainA.GetSimApp().GetIBCKeeper().KVStoreService.OpenKVStore(ctx)
					err := store.Set(ibcmock.TestKey, ibcmock.TestValue)
					suite.Require().NoError(err)

					eventService := suite.chainA.GetSimApp().GetIBCKeeper().EventService
					err = eventService.EventManager(ctx).EmitKV(ibcmock.MockEventType)
					suite.Require().NoError(err)
					return ibcmock.UpgradeVersion, nil
				}
			},
			func(res *channeltypes.MsgChannelUpgradeTryResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(uint64(1), res.UpgradeSequence)

				storeKey := suite.chainA.GetSimApp().GetKey(exported.ModuleName)
				store := suite.chainA.GetContext().KVStore(storeKey)
				suite.Require().Nil(store.Get(ibcmock.TestKey))

				for _, event := range events {
					if event.GetType() == ibcmock.MockEventType {
						suite.Fail("expected application callback events to be discarded")
					}
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			// configure the channel upgrade version on testing endpoints
			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			err = path.EndpointB.UpdateClient()
			suite.Require().NoError(err)

			counterpartySequence := path.EndpointA.GetChannel().UpgradeSequence
			counterpartyUpgrade, found := suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			suite.Require().True(found)

			channelProof, upgradeProof, proofHeight := path.EndpointA.QueryChannelUpgradeProof()

			msg = &channeltypes.MsgChannelUpgradeTry{
				PortId:                        path.EndpointB.ChannelConfig.PortID,
				ChannelId:                     path.EndpointB.ChannelID,
				ProposedUpgradeConnectionHops: []string{ibctesting.FirstConnectionID},
				CounterpartyUpgradeSequence:   counterpartySequence,
				CounterpartyUpgradeFields:     counterpartyUpgrade.Fields,
				ProofChannel:                  channelProof,
				ProofUpgrade:                  upgradeProof,
				ProofHeight:                   proofHeight,
				Signer:                        suite.chainB.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			ctx := suite.chainB.GetContext()
			res, err := suite.chainB.GetSimApp().GetIBCKeeper().ChannelUpgradeTry(ctx, msg)
			events := ctx.EventManager().Events().ToABCIEvents()

			tc.expResult(res, events, err)
		})
	}
}

func (suite *KeeperTestSuite) TestChannelUpgradeAck() {
	var (
		path *ibctesting.Path
		msg  *channeltypes.MsgChannelUpgradeAck
	)

	cases := []struct {
		name      string
		malleate  func()
		expResult func(res *channeltypes.MsgChannelUpgradeAckResponse, events []abci.Event, err error)
	}{
		{
			"success, no pending in-flight packets",
			func() {},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(channeltypes.FLUSHCOMPLETE, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeAck,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"success, pending in-flight packets",
			func() {
				portID := path.EndpointA.ChannelConfig.PortID
				channelID := path.EndpointA.ChannelID
				// Set a dummy packet commitment to simulate in-flight packets
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), portID, channelID, 1, []byte("hash"))
			},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(channeltypes.FLUSHING, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeAck,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"core handler returns error and no upgrade error receipt is written",
			func() {
				// force an error by overriding the channel state to an invalid value
				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
			},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, events []abci.Event, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)
				suite.Require().ErrorIs(err, channeltypes.ErrInvalidChannelState)

				errorReceipt, found := suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().Empty(errorReceipt)
				suite.Require().False(found)

				suite.Require().Empty(events)
			},
		},
		{
			"core handler returns error and writes upgrade error receipt",
			func() {
				// force an upgrade error by modifying the channel upgrade ordering to an incompatible value
				upgrade := path.EndpointA.GetChannelUpgrade()
				upgrade.Fields.Ordering = channeltypes.NONE

				path.EndpointA.SetChannelUpgrade(upgrade)
			},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)

				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.FAILURE, res.Result)

				errorReceipt, found := suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found)
				suite.Require().Equal(uint64(1), errorReceipt.Sequence)

				channel := path.EndpointB.GetChannel()
				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeError,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
						// need to manually insert this because the errorReceipt is a string constant as it is written into state
						sdk.NewAttribute(channeltypes.AttributeKeyErrorReceipt, "expected upgrade ordering (ORDER_NONE_UNSPECIFIED) to match counterparty upgrade ordering (ORDER_UNORDERED): incompatible counterparty upgrade"),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"application callback returns error and error receipt is written",
			func() {
				suite.chainA.GetSimApp().IBCMockModule.IBCApp.OnChanUpgradeAck = func(
					ctx context.Context, portID, channelID, counterpartyVersion string,
				) error {
					// set arbitrary value in store to mock application state changes
					store := suite.chainA.GetSimApp().GetIBCKeeper().KVStoreService.OpenKVStore(ctx)
					err := store.Set(ibcmock.TestKey, ibcmock.TestValue)
					suite.Require().NoError(err)
					return fmt.Errorf("mock app callback failed")
				}
			},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)

				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.FAILURE, res.Result)

				errorReceipt, found := suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found)
				suite.Require().Equal(uint64(1), errorReceipt.Sequence)

				// assert application state changes are not committed
				store := suite.chainA.GetContext().KVStore(suite.chainA.GetSimApp().GetKey(exported.ModuleName))
				suite.Require().False(store.Has([]byte("foo")))

				channel := path.EndpointB.GetChannel()
				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeError,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
						// need to manually insert this because the errorReceipt is a string constant as it is written into state
						sdk.NewAttribute(channeltypes.AttributeKeyErrorReceipt, "mock app callback failed"),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"ibc application does not implement the UpgradeableModule interface",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.EndpointA.ChannelConfig.PortID = ibcmock.MockBlockUpgrade
				path.EndpointB.ChannelConfig.PortID = ibcmock.MockBlockUpgrade

				path.Setup()

				msg.PortId = path.EndpointA.ChannelConfig.PortID
				msg.ChannelId = path.EndpointA.ChannelID
			},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, events []abci.Event, err error) {
				suite.Require().ErrorIs(err, porttypes.ErrInvalidRoute)
				suite.Require().Nil(res)
				suite.Require().Empty(events)
			},
		},
		{
			"application callback returns an upgrade error",
			func() {
				suite.chainA.GetSimApp().IBCMockModule.IBCApp.OnChanUpgradeAck = func(ctx context.Context, portID, channelID, counterpartyVersion string) error {
					return channeltypes.NewUpgradeError(10000000, ibcmock.MockApplicationCallbackError)
				}
			},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, events []abci.Event, err error) {
				suite.Require().Equal(channeltypes.FAILURE, res.Result)
				suite.Require().Equal(uint64(1), path.EndpointA.GetChannel().UpgradeSequence, "application callback upgrade sequence should not be used")
			},
		},
		{
			"ibc application does not commit state changes in callback",
			func() {
				suite.chainA.GetSimApp().IBCMockModule.IBCApp.OnChanUpgradeAck = func(ctx context.Context, portID, channelID, counterpartyVersion string) error {
					store := suite.chainA.GetSimApp().GetIBCKeeper().KVStoreService.OpenKVStore(ctx)
					err := store.Set(ibcmock.TestKey, ibcmock.TestValue)
					suite.Require().NoError(err)

					eventService := suite.chainA.GetSimApp().GetIBCKeeper().EventService
					err = eventService.EventManager(ctx).EmitKV(ibcmock.MockEventType)
					suite.Require().NoError(err)
					return nil
				}
			},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				storeKey := suite.chainA.GetSimApp().GetKey(exported.ModuleName)
				store := suite.chainA.GetContext().KVStore(storeKey)
				suite.Require().Nil(store.Get(ibcmock.TestKey))

				for _, event := range events {
					if event.GetType() == ibcmock.MockEventType {
						suite.Fail("expected application callback events to be discarded")
					}
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			// configure the channel upgrade version on testing endpoints
			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanUpgradeTry()
			suite.Require().NoError(err)

			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			counterpartyUpgrade := path.EndpointB.GetChannelUpgrade()

			channelProof, upgradeProof, proofHeight := path.EndpointB.QueryChannelUpgradeProof()

			msg = &channeltypes.MsgChannelUpgradeAck{
				PortId:              path.EndpointA.ChannelConfig.PortID,
				ChannelId:           path.EndpointA.ChannelID,
				CounterpartyUpgrade: counterpartyUpgrade,
				ProofChannel:        channelProof,
				ProofUpgrade:        upgradeProof,
				ProofHeight:         proofHeight,
				Signer:              suite.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().GetIBCKeeper().ChannelUpgradeAck(ctx, msg)
			events := ctx.EventManager().Events().ToABCIEvents()

			tc.expResult(res, events, err)
		})
	}
}

func (suite *KeeperTestSuite) TestChannelUpgradeConfirm() {
	var (
		path *ibctesting.Path
		msg  *channeltypes.MsgChannelUpgradeConfirm
	)

	cases := []struct {
		name      string
		malleate  func()
		expResult func(res *channeltypes.MsgChannelUpgradeConfirmResponse, events []abci.Event, err error)
	}{
		{
			"success, no pending in-flight packets",
			func() {},
			func(res *channeltypes.MsgChannelUpgradeConfirmResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)

				channel := path.EndpointB.GetChannel()
				suite.Require().Equal(channeltypes.OPEN, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeConfirm,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelState, channeltypes.FLUSHCOMPLETE.String()),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeOpen,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelState, channeltypes.OPEN.String()),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"success, pending in-flight packets on init chain",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion
				path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion

				err := path.EndpointA.ChanUpgradeInit()
				suite.Require().NoError(err)

				err = path.EndpointB.ChanUpgradeTry()
				suite.Require().NoError(err)

				seq, err := path.EndpointA.SendPacket(path.EndpointB.Chain.GetTimeoutHeight(), 0, ibctesting.MockPacketData)
				suite.Require().Equal(uint64(1), seq)
				suite.Require().NoError(err)

				err = path.EndpointA.ChanUpgradeAck()
				suite.Require().NoError(err)

				err = path.EndpointB.UpdateClient()
				suite.Require().NoError(err)

				counterpartyChannelState := path.EndpointA.GetChannel().State
				counterpartyUpgrade := path.EndpointA.GetChannelUpgrade()

				channelProof, upgradeProof, proofHeight := path.EndpointA.QueryChannelUpgradeProof()

				msg = &channeltypes.MsgChannelUpgradeConfirm{
					PortId:                   path.EndpointB.ChannelConfig.PortID,
					ChannelId:                path.EndpointB.ChannelID,
					CounterpartyChannelState: counterpartyChannelState,
					CounterpartyUpgrade:      counterpartyUpgrade,
					ProofChannel:             channelProof,
					ProofUpgrade:             upgradeProof,
					ProofHeight:              proofHeight,
					Signer:                   suite.chainA.SenderAccount.GetAddress().String(),
				}
			},
			func(res *channeltypes.MsgChannelUpgradeConfirmResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(channeltypes.FLUSHING, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)

				channel = path.EndpointB.GetChannel()
				suite.Require().Equal(channeltypes.FLUSHCOMPLETE, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeConfirm,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelState, channeltypes.FLUSHCOMPLETE.String()),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"success, pending in-flight packets on try chain",
			func() {
				portID, channelID := path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID
				suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.SetPacketCommitment(suite.chainB.GetContext(), portID, channelID, 1, []byte("hash"))
			},
			func(res *channeltypes.MsgChannelUpgradeConfirmResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)

				channel := path.EndpointB.GetChannel()
				suite.Require().Equal(channeltypes.FLUSHING, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeConfirm,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelState, channeltypes.FLUSHING.String()),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"core handler returns error and no upgrade error receipt is written",
			func() {
				// force an error by overriding the channel state to an invalid value
				path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
			},
			func(res *channeltypes.MsgChannelUpgradeConfirmResponse, events []abci.Event, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)
				suite.Require().ErrorIs(err, channeltypes.ErrInvalidChannelState)

				errorReceipt, found := suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				suite.Require().Empty(errorReceipt)
				suite.Require().False(found)

				suite.Require().Empty(events)
			},
		},
		{
			"core handler returns error and writes upgrade error receipt",
			func() {
				// force an upgrade error by modifying the counterparty channel upgrade timeout to be elapsed
				upgrade := path.EndpointA.GetChannelUpgrade()
				upgrade.Timeout = channeltypes.NewTimeout(clienttypes.ZeroHeight(), uint64(path.EndpointB.Chain.ProposedHeader.Time.UnixNano()))

				path.EndpointA.SetChannelUpgrade(upgrade)

				suite.coordinator.CommitBlock(suite.chainA)

				err := path.EndpointB.UpdateClient()
				suite.Require().NoError(err)

				channelProof, upgradeProof, proofHeight := path.EndpointA.QueryChannelUpgradeProof()

				msg.CounterpartyUpgrade = upgrade
				msg.ProofChannel = channelProof
				msg.ProofUpgrade = upgradeProof
				msg.ProofHeight = proofHeight
			},
			func(res *channeltypes.MsgChannelUpgradeConfirmResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)

				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.FAILURE, res.Result)

				errorReceipt, found := suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				suite.Require().True(found)
				suite.Require().Equal(uint64(1), errorReceipt.Sequence)

				channel := path.EndpointB.GetChannel()

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeError,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
						// need to manually insert this because the errorReceipt is a string constant as it is written into state
						sdk.NewAttribute(channeltypes.AttributeKeyErrorReceipt, "counterparty upgrade timeout elapsed: current timestamp: 1578269010000000000, timeout timestamp 1578268995000000000: timeout elapsed"),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"ibc application does not implement the UpgradeableModule interface",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.EndpointA.ChannelConfig.PortID = ibcmock.MockBlockUpgrade
				path.EndpointB.ChannelConfig.PortID = ibcmock.MockBlockUpgrade

				path.Setup()

				msg.PortId = path.EndpointB.ChannelConfig.PortID
				msg.ChannelId = path.EndpointB.ChannelID
			},
			func(res *channeltypes.MsgChannelUpgradeConfirmResponse, events []abci.Event, err error) {
				suite.Require().ErrorIs(err, porttypes.ErrInvalidRoute)
				suite.Require().Nil(res)

				suite.Require().Empty(events)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			// configure the channel upgrade version on testing endpoints
			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanUpgradeTry()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanUpgradeAck()
			suite.Require().NoError(err)

			err = path.EndpointB.UpdateClient()
			suite.Require().NoError(err)

			counterpartyChannelState := path.EndpointA.GetChannel().State
			counterpartyUpgrade := path.EndpointA.GetChannelUpgrade()

			channelProof, upgradeProof, proofHeight := path.EndpointA.QueryChannelUpgradeProof()

			msg = &channeltypes.MsgChannelUpgradeConfirm{
				PortId:                   path.EndpointB.ChannelConfig.PortID,
				ChannelId:                path.EndpointB.ChannelID,
				CounterpartyChannelState: counterpartyChannelState,
				CounterpartyUpgrade:      counterpartyUpgrade,
				ProofChannel:             channelProof,
				ProofUpgrade:             upgradeProof,
				ProofHeight:              proofHeight,
				Signer:                   suite.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			ctx := suite.chainB.GetContext()
			res, err := suite.chainB.GetSimApp().GetIBCKeeper().ChannelUpgradeConfirm(ctx, msg)
			events := ctx.EventManager().Events().ToABCIEvents()

			tc.expResult(res, events, err)
		})
	}
}

func (suite *KeeperTestSuite) TestChannelUpgradeOpen() {
	var (
		path *ibctesting.Path
		msg  *channeltypes.MsgChannelUpgradeOpen
	)

	cases := []struct {
		name      string
		malleate  func()
		expResult func(res *channeltypes.MsgChannelUpgradeOpenResponse, events []abci.Event, err error)
	}{
		{
			"success",
			func() {},
			func(res *channeltypes.MsgChannelUpgradeOpenResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(channeltypes.OPEN, channel.State)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeOpen,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelState, channeltypes.OPEN.String()),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"success with counterparty at greater upgrade sequence",
			func() {
				// create reason to upgrade
				path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion + "additional upgrade"

				err := path.EndpointB.ChanUpgradeInit()
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				counterpartyChannel := path.EndpointB.GetChannel()
				channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				channelProof, proofHeight := path.EndpointB.QueryProof(channelKey)

				msg.ProofChannel = channelProof
				msg.ProofHeight = proofHeight
				msg.CounterpartyUpgradeSequence = counterpartyChannel.UpgradeSequence
			},
			func(res *channeltypes.MsgChannelUpgradeOpenResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(channeltypes.OPEN, channel.State)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeOpen,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelState, channeltypes.OPEN.String()),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"core handler fails",
			func() {
				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.FLUSHING })
			},
			func(res *channeltypes.MsgChannelUpgradeOpenResponse, events []abci.Event, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)

				suite.Require().ErrorIs(err, channeltypes.ErrInvalidChannelState)
				suite.Require().Empty(events)
			},
		},
		{
			"ibc application does not implement the UpgradeableModule interface",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.EndpointA.ChannelConfig.PortID = ibcmock.MockBlockUpgrade
				path.EndpointB.ChannelConfig.PortID = ibcmock.MockBlockUpgrade

				path.Setup()

				msg.PortId = path.EndpointA.ChannelConfig.PortID
				msg.ChannelId = path.EndpointA.ChannelID
			},
			func(res *channeltypes.MsgChannelUpgradeOpenResponse, events []abci.Event, err error) {
				suite.Require().ErrorIs(err, porttypes.ErrInvalidRoute)
				suite.Require().Nil(res)

				suite.Require().Empty(events)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			// configure the channel upgrade version on testing endpoints
			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanUpgradeTry()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanUpgradeAck()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanUpgradeConfirm()
			suite.Require().NoError(err)

			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			counterpartyChannel := path.EndpointB.GetChannel()
			channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			channelProof, proofHeight := path.EndpointB.QueryProof(channelKey)

			msg = &channeltypes.MsgChannelUpgradeOpen{
				PortId:                      path.EndpointA.ChannelConfig.PortID,
				ChannelId:                   path.EndpointA.ChannelID,
				CounterpartyChannelState:    counterpartyChannel.State,
				CounterpartyUpgradeSequence: counterpartyChannel.UpgradeSequence,
				ProofChannel:                channelProof,
				ProofHeight:                 proofHeight,
				Signer:                      suite.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().GetIBCKeeper().ChannelUpgradeOpen(ctx, msg)
			events := ctx.EventManager().Events().ToABCIEvents()

			tc.expResult(res, events, err)
		})
	}
}

func (suite *KeeperTestSuite) TestChannelUpgradeCancel() {
	var (
		path *ibctesting.Path
		msg  *channeltypes.MsgChannelUpgradeCancel
	)

	cases := []struct {
		name      string
		malleate  func()
		expResult func(res *channeltypes.MsgChannelUpgradeCancelResponse, events []abci.Event, err error)
	}{
		{
			"success: keeper is not authority, valid error receipt so channel changed to match error receipt seq",
			func() {},
			func(res *channeltypes.MsgChannelUpgradeCancelResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				channel := path.EndpointA.GetChannel()
				// Channel state should be reverted back to open.
				suite.Require().Equal(channeltypes.OPEN, channel.State)
				// Upgrade sequence should be changed to match sequence on error receipt.
				suite.Require().Equal(uint64(2), channel.UpgradeSequence)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeCancel,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeError,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
						sdk.NewAttribute(channeltypes.AttributeKeyErrorReceipt, "invalid upgrade"),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"success: keeper is authority & channel state in FLUSHING, so error receipt is ignored and channel is restored to initial upgrade sequence",
			func() {
				msg.Signer = suite.chainA.App.GetIBCKeeper().GetAuthority()

				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) {
					channel.State = channeltypes.FLUSHING
					channel.UpgradeSequence = uint64(3)
				})
			},
			func(res *channeltypes.MsgChannelUpgradeCancelResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				channel := path.EndpointA.GetChannel()
				// Channel state should be reverted back to open.
				suite.Require().Equal(channeltypes.OPEN, channel.State)
				// Upgrade sequence should be changed to match initial upgrade sequence.
				suite.Require().Equal(uint64(3), channel.UpgradeSequence)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeCancel,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeError,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
						// need to manually insert this because the errorReceipt is a string constant as it is written into state
						sdk.NewAttribute(channeltypes.AttributeKeyErrorReceipt, "invalid upgrade"),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"success: keeper is authority & channel state in FLUSHING, can be cancelled even with invalid error receipt",
			func() {
				msg.Signer = suite.chainA.App.GetIBCKeeper().GetAuthority()
				msg.ProofErrorReceipt = []byte("invalid proof")

				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) {
					channel.State = channeltypes.FLUSHING
					channel.UpgradeSequence = uint64(1)
				})
			},
			func(res *channeltypes.MsgChannelUpgradeCancelResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				channel := path.EndpointA.GetChannel()
				// Channel state should be reverted back to open.
				suite.Require().Equal(channeltypes.OPEN, channel.State)
				// Upgrade sequence should be changed to match initial upgrade sequence.
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeCancel,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeError,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
						// need to manually insert this because the errorReceipt is a string constant as it is written into state
						sdk.NewAttribute(channeltypes.AttributeKeyErrorReceipt, "invalid upgrade"),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"success: keeper is authority & channel state in FLUSHING, can be cancelled even with empty error receipt",
			func() {
				msg.Signer = suite.chainA.App.GetIBCKeeper().GetAuthority()
				msg.ProofErrorReceipt = nil

				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) {
					channel.State = channeltypes.FLUSHING
					channel.UpgradeSequence = uint64(1)
				})
			},
			func(res *channeltypes.MsgChannelUpgradeCancelResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				channel := path.EndpointA.GetChannel()
				// Channel state should be reverted back to open.
				suite.Require().Equal(channeltypes.OPEN, channel.State)
				// Upgrade sequence should be changed to match initial upgrade sequence.
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeCancel,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeError,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
						// need to manually insert this because the errorReceipt is a string constant as it is written into state
						sdk.NewAttribute(channeltypes.AttributeKeyErrorReceipt, "invalid upgrade"),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"success: keeper is authority but channel state in FLUSHCOMPLETE, requires valid error receipt",
			func() {
				msg.Signer = suite.chainA.App.GetIBCKeeper().GetAuthority()

				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) {
					channel.State = channeltypes.FLUSHCOMPLETE
					channel.UpgradeSequence = uint64(2) // When in FLUSHCOMPLETE the sequence of the error receipt and the channel must match
				})
			},
			func(res *channeltypes.MsgChannelUpgradeCancelResponse, events []abci.Event, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				channel := path.EndpointA.GetChannel()
				// Channel state should be reverted back to open.
				suite.Require().Equal(channeltypes.OPEN, channel.State)
				// Upgrade sequence should not be changed.
				suite.Require().Equal(uint64(2), channel.UpgradeSequence)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeCancel,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeError,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
						// need to manually insert this because the errorReceipt is a string constant as it is written into state
						sdk.NewAttribute(channeltypes.AttributeKeyErrorReceipt, "invalid upgrade"),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(
							sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"failure: keeper is authority and channel state in FLUSHCOMPLETE, but error receipt and channel upgrade sequences do not match",
			func() {
				msg.Signer = suite.chainA.App.GetIBCKeeper().GetAuthority()

				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.FLUSHCOMPLETE })
			},
			func(res *channeltypes.MsgChannelUpgradeCancelResponse, events []abci.Event, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)
				suite.Require().ErrorIs(err, channeltypes.ErrInvalidUpgradeSequence)

				channel := path.EndpointA.GetChannel()
				// Channel state should not be reverted back to open.
				suite.Require().Equal(channeltypes.FLUSHCOMPLETE, channel.State)
				// Upgrade sequence should not be changed.
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)
			},
		},
		{
			"core handler fails: invalid proof",
			func() {
				msg.ProofErrorReceipt = []byte("invalid proof")

				// Force set to STATE_FLUSHCOMPLETE to check that state is not changed.
				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) {
					channel.State = channeltypes.FLUSHCOMPLETE
					channel.UpgradeSequence = uint64(2) // When in FLUSHCOMPLETE the sequence of the error receipt and the channel must match
				})
			},
			func(res *channeltypes.MsgChannelUpgradeCancelResponse, events []abci.Event, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)

				channel := path.EndpointA.GetChannel()
				suite.Require().ErrorIs(err, commitmenttypes.ErrInvalidProof)
				// Channel state should not be changed.
				suite.Require().Equal(channeltypes.FLUSHCOMPLETE, channel.State)
				// Upgrade sequence should not be changed.
				suite.Require().Equal(uint64(2), channel.UpgradeSequence)

				suite.Require().Empty(events)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			// configure the channel upgrade version on testing endpoints
			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			// cause the upgrade to fail on chain b so an error receipt is written.
			// if the counterparty (chain A) upgrade sequence is less than the current sequence, (chain B)
			// an upgrade error will be returned by chain B during ChanUpgradeTry.
			path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.UpgradeSequence = uint64(1) })
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.UpgradeSequence = uint64(2) })

			suite.Require().NoError(path.EndpointA.UpdateClient())
			suite.Require().NoError(path.EndpointB.UpdateClient())

			suite.Require().NoError(path.EndpointB.ChanUpgradeTry())

			suite.Require().NoError(path.EndpointA.UpdateClient())

			upgradeErrorReceiptKey := host.ChannelUpgradeErrorKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			errorReceiptProof, proofHeight := path.EndpointB.QueryProof(upgradeErrorReceiptKey)

			errorReceipt, ok := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			suite.Require().True(ok)

			msg = &channeltypes.MsgChannelUpgradeCancel{
				PortId:            path.EndpointA.ChannelConfig.PortID,
				ChannelId:         path.EndpointA.ChannelID,
				ErrorReceipt:      errorReceipt,
				ProofErrorReceipt: errorReceiptProof,
				ProofHeight:       proofHeight,
				Signer:            suite.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().GetIBCKeeper().ChannelUpgradeCancel(ctx, msg)
			events := ctx.EventManager().Events().ToABCIEvents()

			tc.expResult(res, events, err)
		})
	}
}

func (suite *KeeperTestSuite) TestChannelUpgradeTimeout() {
	var (
		path *ibctesting.Path
		msg  *channeltypes.MsgChannelUpgradeTimeout
	)

	timeoutUpgrade := func() {
		upgrade := path.EndpointA.GetProposedUpgrade()
		upgrade.Timeout = channeltypes.NewTimeout(clienttypes.ZeroHeight(), 1)
		path.EndpointA.SetChannelUpgrade(upgrade)
		suite.Require().NoError(path.EndpointB.UpdateClient())
	}

	cases := []struct {
		name      string
		malleate  func()
		expResult func(res *channeltypes.MsgChannelUpgradeTimeoutResponse, events []abci.Event, err error)
	}{
		{
			"success",
			func() {
				timeoutUpgrade()

				suite.Require().NoError(path.EndpointA.UpdateClient())

				channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				channelProof, proofHeight := path.EndpointB.QueryProof(channelKey)

				msg.ProofChannel = channelProof
				msg.ProofHeight = proofHeight
			},
			func(res *channeltypes.MsgChannelUpgradeTimeoutResponse, events []abci.Event, err error) {
				channel := path.EndpointA.GetChannel()

				suite.Require().Equalf(channeltypes.OPEN, channel.State, "channel state should be %s", channeltypes.OPEN.String())

				_, found := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().False(found, "channel upgrade should be nil")

				suite.Require().NotNil(res)
				suite.Require().NoError(err)

				errorReceipt, found := suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found)
				suite.Require().Equal(uint64(1), errorReceipt.Sequence)

				// use the timeout we set in the malleate function
				timeout := channeltypes.NewTimeout(clienttypes.ZeroHeight(), 1)

				expEvents := sdk.Events{
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeTimeout,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeTimeoutHeight, timeout.Height.String()),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeTimeoutTimestamp, fmt.Sprintf("%d", timeout.Timestamp)),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
					),
					sdk.NewEvent(
						channeltypes.EventTypeChannelUpgradeError,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(channeltypes.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(channeltypes.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
						// need to manually insert this because the errorReceipt is a string constant as it is written into state
						sdk.NewAttribute(channeltypes.AttributeKeyErrorReceipt, "upgrade timed-out"),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, channeltypes.AttributeValueCategory),
					),
				}.ToABCIEvents()
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			},
		},
		{
			"core handler fails: invalid proof",
			func() {
				timeoutUpgrade()

				suite.Require().NoError(path.EndpointA.UpdateClient())

				_, _, proofHeight := path.EndpointB.QueryChannelUpgradeProof()

				msg.ProofHeight = proofHeight
				msg.ProofChannel = []byte("invalid proof")
			},
			func(res *channeltypes.MsgChannelUpgradeTimeoutResponse, events []abci.Event, err error) {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, commitmenttypes.ErrInvalidProof)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equalf(channeltypes.FLUSHCOMPLETE, channel.State, "channel state should be %s", channeltypes.OPEN)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence, "channel upgrade sequence should not incremented")

				_, found := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found, "channel upgrade should not be nil")

				suite.Require().Empty(events)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion

			suite.Require().NoError(path.EndpointA.ChanUpgradeInit())
			suite.Require().NoError(path.EndpointB.ChanUpgradeTry())
			suite.Require().NoError(path.EndpointA.ChanUpgradeAck())

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channelProof, proofHeight := path.EndpointB.QueryProof(channelKey)

			msg = &channeltypes.MsgChannelUpgradeTimeout{
				PortId:              path.EndpointA.ChannelConfig.PortID,
				ChannelId:           path.EndpointA.ChannelID,
				CounterpartyChannel: path.EndpointB.GetChannel(),
				ProofChannel:        channelProof,
				ProofHeight:         proofHeight,
				Signer:              suite.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().GetIBCKeeper().ChannelUpgradeTimeout(ctx, msg)
			events := ctx.EventManager().Events().ToABCIEvents()

			tc.expResult(res, events, err)
		})
	}
}

// TestIBCSoftwareUpgrade tests the IBCSoftwareUpgrade rpc handler
func (suite *KeeperTestSuite) TestIBCSoftwareUpgrade() {
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
				msg.Signer = suite.chainA.SenderAccount.GetAddress().String()
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
		tc := tc
		suite.Run(tc.name, func() {
			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()
			validAuthority := suite.chainA.App.GetIBCKeeper().GetAuthority()
			plan := upgradetypes.Plan{
				Name:   "upgrade IBC clients",
				Height: 1000,
			}
			// update trusting period
			clientState, ok := path.EndpointB.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			clientState.TrustingPeriod += 100

			var err error
			msg, err = clienttypes.NewMsgIBCSoftwareUpgrade(
				validAuthority,
				plan,
				clientState,
			)

			suite.Require().NoError(err)

			tc.malleate()

			_, err = suite.chainA.App.GetIBCKeeper().IBCSoftwareUpgrade(suite.chainA.GetContext(), msg)

			if tc.expError == nil {
				suite.Require().NoError(err)
				// upgrade plan is stored
				storedPlan, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradePlan(suite.chainA.GetContext())
				suite.Require().NoError(err)
				suite.Require().Equal(plan, storedPlan)

				// upgraded client state is stored
				bz, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(suite.chainA.GetContext(), plan.Height)
				suite.Require().NoError(err)
				upgradedClientState, err := clienttypes.UnmarshalClientState(suite.chainA.App.AppCodec(), bz)
				suite.Require().NoError(err)
				suite.Require().Equal(clientState.ZeroCustomFields(), upgradedClientState)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

// TestUpdateClientParams tests the UpdateClientParams rpc handler
func (suite *KeeperTestSuite) TestUpdateClientParams() {
	signer := suite.chainA.App.GetIBCKeeper().GetAuthority()
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
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			_, err := suite.chainA.App.GetIBCKeeper().UpdateClientParams(suite.chainA.GetContext(), tc.msg)
			if tc.expError == nil {
				suite.Require().NoError(err)
				p := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(suite.chainA.GetContext())
				suite.Require().Equal(tc.msg.Params, p)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

// TestUpdateConnectionParams tests the UpdateConnectionParams rpc handler
func (suite *KeeperTestSuite) TestUpdateConnectionParams() {
	signer := suite.chainA.App.GetIBCKeeper().GetAuthority()
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
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			_, err := suite.chainA.App.GetIBCKeeper().UpdateConnectionParams(suite.chainA.GetContext(), tc.msg)
			if tc.expErr == nil {
				suite.Require().NoError(err)
				p := suite.chainA.App.GetIBCKeeper().ConnectionKeeper.GetParams(suite.chainA.GetContext())
				suite.Require().Equal(tc.msg.Params, p)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErr.Error())
			}
		})
	}
}

// TestUpdateChannelParams tests the UpdateChannelParams rpc handler
func (suite *KeeperTestSuite) TestUpdateChannelParams() {
	authority := suite.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name     string
		msg      *channeltypes.MsgUpdateParams
		expError error
	}{
		{
			"success: valid authority and default params",
			channeltypes.NewMsgUpdateChannelParams(authority, channeltypes.DefaultParams()),
			nil,
		},
		{
			"failure: malformed authority address",
			channeltypes.NewMsgUpdateChannelParams(ibctesting.InvalidID, channeltypes.DefaultParams()),
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: empty authority address",
			channeltypes.NewMsgUpdateChannelParams("", channeltypes.DefaultParams()),
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: empty signer",
			channeltypes.NewMsgUpdateChannelParams("", channeltypes.DefaultParams()),
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: unauthorized authority address",
			channeltypes.NewMsgUpdateChannelParams(ibctesting.TestAccAddress, channeltypes.DefaultParams()),
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			resp, err := suite.chainA.App.GetIBCKeeper().UpdateChannelParams(suite.chainA.GetContext(), tc.msg)
			if tc.expError == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(resp)
				p := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetParams(suite.chainA.GetContext())
				suite.Require().Equal(tc.msg.Params, p)
			} else {
				suite.Require().Nil(resp)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestPruneAcknowledgements() {
	var msg *channeltypes.MsgPruneAcknowledgements

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"failure: core keeper function fails, pruning sequence end not found",
			func() {
				msg.PortId = "portidone"
			},
			channeltypes.ErrRecvStartSequenceNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			// configure the channel upgrade version on testing endpoints
			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanUpgradeTry()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanUpgradeAck()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanUpgradeConfirm()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanUpgradeOpen()
			suite.Require().NoError(err)

			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			msg = channeltypes.NewMsgPruneAcknowledgements(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				10,
				suite.chainA.SenderAccount.GetAddress().String(),
			)

			tc.malleate()

			resp, err := suite.chainA.App.GetIBCKeeper().PruneAcknowledgements(suite.chainA.GetContext(), msg)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(resp)
				suite.Require().Equal(uint64(0), resp.TotalPrunedSequences)
				suite.Require().Equal(uint64(0), resp.TotalRemainingSequences)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(resp)
			}
		})
	}
}
