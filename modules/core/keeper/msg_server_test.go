package keeper_test

import (
	"errors"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
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
		async  bool // indicate no ack written
	)

	testCases := []struct {
		name      string
		malleate  func()
		expPass   bool
		expRevert bool
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, false},
		{"success: UNORDERED", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, false},
		{"success: UNORDERED out of order packet", func() {
			// setup uses an UNORDERED channel
			suite.coordinator.Setup(path)

			// attempts to receive packet with sequence 10 without receiving packet with sequence 1
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}
		}, true, false},
		{"success: OnRecvPacket callback returns revert=true", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockFailPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockFailPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, true},
		{"success: ORDERED - async acknowledgement", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)
			async = true

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibcmock.MockAsyncPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibcmock.MockAsyncPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, false},
		{"success: UNORDERED - async acknowledgement", func() {
			suite.coordinator.Setup(path)
			async = true

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibcmock.MockAsyncPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibcmock.MockAsyncPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, false},
		{"failure: ORDERED out of order packet", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			// attempts to receive packet with sequence 10 without receiving packet with sequence 1
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}
		}, false, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)
		}, false, false},
		{"packet not sent", func() {
			suite.coordinator.Setup(path)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, false, false},
		{"successful no-op: ORDERED - packet already received (replay)", func() {
			// mock will panic if application callback is called twice on the same packet
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, true, false},
		{"successful no-op: UNORDERED - packet already received (replay)", func() {
			// mock will panic if application callback is called twice on the same packet
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, true, false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			async = false     // reset
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

			_, err := keeper.Keeper.RecvPacket(*suite.chainB.App.GetIBCKeeper(), suite.chainB.GetContext(), msg)

			if tc.expPass {
				suite.Require().NoError(err)

				// replay should not fail since it will be treated as a no-op
				_, err := keeper.Keeper.RecvPacket(*suite.chainB.App.GetIBCKeeper(), suite.chainB.GetContext(), msg)
				suite.Require().NoError(err)

				// check that callback state was handled correctly
				_, exists := suite.chainB.GetSimApp().ScopedIBCMockKeeper.GetCapability(suite.chainB.GetContext(), ibcmock.GetMockRecvCanaryCapabilityName(packet))
				if tc.expRevert {
					suite.Require().False(exists, "capability exists in store even after callback reverted")
				} else {
					suite.Require().True(exists, "callback state not persisted when revert is false")
				}

				// verify if ack was written
				ack, found := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

				if async {
					suite.Require().Nil(ack)
					suite.Require().False(found)

				} else {
					suite.Require().NotNil(ack)
					suite.Require().True(found)
				}
			} else {
				suite.Require().Error(err)
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
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupClients(subjectPath)
			subject := subjectPath.EndpointA.ClientID
			subjectClientState := suite.chainA.GetClientState(subject)

			substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupClients(substitutePath)
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

			_, err = keeper.Keeper.RecoverClient(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				// Assert that client status is now Active
				clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)
				tmClientState := subjectPath.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().Equal(tmClientState.Status(suite.chainA.GetContext(), clientStore, suite.chainA.App.AppCodec()), exported.Active)
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
		expPass  bool
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, true},
		{"success: UNORDERED", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, true},
		{"success: UNORDERED acknowledge out of order packet", func() {
			// setup uses an UNORDERED channel
			suite.coordinator.Setup(path)

			// attempts to acknowledge ack with sequence 10 without acknowledging ack with sequence 1 (removing packet commitment)
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)
			}
		}, true},
		{"failure: ORDERED acknowledge out of order packet", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			// attempts to acknowledge ack with sequence 10 without acknowledging ack with sequence 1 (removing packet commitment
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)
			}
		}, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)
		}, false},
		{"packet not received", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, false},
		{"successful no-op: ORDERED - packet already acknowledged (replay)", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
			suite.Require().NoError(err)
		}, true},
		{"successful no-op: UNORDERED - packet already acknowledged (replay)", func() {
			suite.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
			suite.Require().NoError(err)
		}, true},
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

			_, err := keeper.Keeper.Acknowledgement(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)

			if tc.expPass {
				suite.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				suite.Require().False(has)

				// replay should not error as it is treated as a no-op
				_, err := keeper.Keeper.Acknowledgement(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
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
		expPass  bool
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

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
		}, true},
		{"success: UNORDERED", func() {
			suite.coordinator.Setup(path)

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
		}, true},
		{"success: UNORDERED timeout out of order packet", func() {
			// setup uses an UNORDERED channel
			suite.coordinator.Setup(path)

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
		}, true},
		{"success: ORDERED timeout out of order packet", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

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
		}, true},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, false},
		{"successful no-op: UNORDERED - packet not sent", func() {
			suite.coordinator.Setup(path)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 1), 0)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, true},
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

			_, err := keeper.Keeper.Timeout(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)

			if tc.expPass {
				suite.Require().NoError(err)

				// replay should not return an error as it is treated as a no-op
				_, err := keeper.Keeper.Timeout(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				suite.Require().False(has)

			} else {
				suite.Require().Error(err)
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
		packet    channeltypes.Packet
		packetKey []byte
		path      *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())

			// close counterparty channel
			err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
			suite.Require().NoError(err)
		}, true},
		{"success: UNORDERED", func() {
			suite.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
			suite.Require().NoError(err)
		}, true},
		{"success: UNORDERED timeout out of order packet", func() {
			// setup uses an UNORDERED channel
			suite.coordinator.Setup(path)

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
			err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
			suite.Require().NoError(err)
		}, true},
		{"success: ORDERED timeout out of order packet", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

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
			err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
			suite.Require().NoError(err)
		}, true},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, false},
		{"successful no-op: UNORDERED - packet not sent", func() {
			suite.coordinator.Setup(path)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 1), 0)
			packetKey = host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			err := path.EndpointB.SetChannelState(channeltypes.CLOSED)
			suite.Require().NoError(err)
		}, true},
		{"ORDERED: channel not closed", func() {
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			proof, proofHeight := suite.chainB.QueryProof(packetKey)

			channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			proofClosed, _ := suite.chainB.QueryProof(channelKey)

			msg := channeltypes.NewMsgTimeoutOnClose(packet, 1, proof, proofClosed, proofHeight, suite.chainA.SenderAccount.GetAddress().String())

			_, err := keeper.Keeper.TimeoutOnClose(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)

			if tc.expPass {
				suite.Require().NoError(err)

				// replay should not return an error as it will be treated as a no-op
				_, err := keeper.Keeper.TimeoutOnClose(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				suite.Require().False(has)

			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUpgradeClient() {
	var (
		path              *ibctesting.Path
		newChainID        string
		newClientHeight   clienttypes.Height
		upgradedClient    exported.ClientState
		upgradedConsState exported.ConsensusState
		lastHeight        exported.Height
		msg               *clienttypes.MsgUpgradeClient
	)
	cases := []struct {
		name    string
		setup   func()
		expPass bool
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

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)

				proofUpgradeClient, _ := suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ := suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())

				msg, err = clienttypes.NewMsgUpgradeClient(path.EndpointA.ClientID, upgradedClient, upgradedConsState,
					proofUpgradeClient, proofUpgradedConsState, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			expPass: true,
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
			expPass: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		path = ibctesting.NewPath(suite.chainA, suite.chainB)
		suite.coordinator.SetupClients(path)

		var err error
		clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
		revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

		newChainID, err = clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
		suite.Require().NoError(err)

		newClientHeight = clienttypes.NewHeight(revisionNumber+1, clientState.GetLatestHeight().GetRevisionHeight()+1)

		tc.setup()

		_, err = keeper.Keeper.UpgradeClient(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)

		if tc.expPass {
			suite.Require().NoError(err, "upgrade handler failed on valid case: %s", tc.name)
			newClient, ok := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
			suite.Require().True(ok)
			newChainSpecifiedClient := newClient.ZeroCustomFields()
			suite.Require().Equal(upgradedClient, newChainSpecifiedClient)
		} else {
			suite.Require().Error(err, "upgrade handler passed on invalid case: %s", tc.name)
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
		expResult func(res *channeltypes.MsgChannelUpgradeInitResponse, err error)
		expEvents bool
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
			func(res *channeltypes.MsgChannelUpgradeInitResponse, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(uint64(1), res.UpgradeSequence)
			},
			true,
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
			func(res *channeltypes.MsgChannelUpgradeInitResponse, err error) {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, ibcerrors.ErrUnauthorized.Error())
				suite.Require().Nil(res)
			},
			false,
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			// configure the channel upgrade version on testing endpoints
			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().GetIBCKeeper().ChannelUpgradeInit(ctx, msg)
			tc.expResult(res, err)

			if tc.expEvents {
				events := ctx.EventManager().Events().ToABCIEvents()
				expEvents := ibctesting.EventsMap{
					types.EventTypeChannelUpgradeInit: {
						types.AttributeKeyPortID:                path.EndpointA.ChannelConfig.PortID,
						types.AttributeKeyChannelID:             path.EndpointA.ChannelID,
						types.AttributeCounterpartyPortID:       path.EndpointB.ChannelConfig.PortID,
						types.AttributeCounterpartyChannelID:    path.EndpointB.ChannelID,
						types.AttributeKeyUpgradeConnectionHops: path.EndpointA.GetProposedUpgrade().Fields.ConnectionHops[0],
						types.AttributeKeyUpgradeVersion:        path.EndpointA.GetProposedUpgrade().Fields.Version,
						types.AttributeKeyUpgradeOrdering:       path.EndpointA.GetProposedUpgrade().Fields.Ordering.String(),
						types.AttributeKeyUpgradeSequence:       fmt.Sprintf("%d", path.EndpointA.GetChannel().UpgradeSequence),
					},
					sdk.EventTypeMessage: {
						sdk.AttributeKeyModule: types.AttributeValueCategory,
					},
				}
				ibctesting.AssertEventsLegacy(&suite.Suite, expEvents, events)
			}
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
		expResult func(res *channeltypes.MsgChannelUpgradeTryResponse, err error)
		expEvents bool
	}{
		{
			"success",
			func() {},
			func(res *channeltypes.MsgChannelUpgradeTryResponse, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)

				channel := path.EndpointB.GetChannel()
				suite.Require().Equal(channeltypes.FLUSHING, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)
			},
			true,
		},
		{
			"module capability not found",
			func() {
				msg.PortId = "invalid-port"
				msg.ChannelId = "invalid-channel"
			},
			func(res *channeltypes.MsgChannelUpgradeTryResponse, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)

				suite.Require().ErrorIs(err, capabilitytypes.ErrCapabilityNotFound)
			},
			false,
		},
		{
			"unsynchronized upgrade sequence writes upgrade error receipt",
			func() {
				channel := path.EndpointB.GetChannel()
				channel.UpgradeSequence = 99

				path.EndpointB.SetChannel(channel)
			},
			func(res *channeltypes.MsgChannelUpgradeTryResponse, err error) {
				suite.Require().NoError(err)

				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.FAILURE, res.Result)

				errorReceipt, found := suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				suite.Require().True(found)
				suite.Require().Equal(uint64(99), errorReceipt.Sequence)
			},
			false,
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

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

			proofChannel, proofUpgrade, proofHeight := path.EndpointA.QueryChannelUpgradeProof()

			msg = &channeltypes.MsgChannelUpgradeTry{
				PortId:                        path.EndpointB.ChannelConfig.PortID,
				ChannelId:                     path.EndpointB.ChannelID,
				ProposedUpgradeConnectionHops: []string{ibctesting.FirstConnectionID},
				CounterpartyUpgradeSequence:   counterpartySequence,
				CounterpartyUpgradeFields:     counterpartyUpgrade.Fields,
				ProofChannel:                  proofChannel,
				ProofUpgrade:                  proofUpgrade,
				ProofHeight:                   proofHeight,
				Signer:                        suite.chainB.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			ctx := suite.chainB.GetContext()
			res, err := suite.chainB.GetSimApp().GetIBCKeeper().ChannelUpgradeTry(ctx, msg)
			tc.expResult(res, err)

			if tc.expEvents {
				events := ctx.EventManager().Events().ToABCIEvents()
				expEvents := ibctesting.EventsMap{
					types.EventTypeChannelUpgradeTry: {
						types.AttributeKeyPortID:                path.EndpointB.ChannelConfig.PortID,
						types.AttributeKeyChannelID:             path.EndpointB.ChannelID,
						types.AttributeCounterpartyPortID:       path.EndpointA.ChannelConfig.PortID,
						types.AttributeCounterpartyChannelID:    path.EndpointA.ChannelID,
						types.AttributeKeyUpgradeConnectionHops: path.EndpointB.GetProposedUpgrade().Fields.ConnectionHops[0],
						types.AttributeKeyUpgradeVersion:        path.EndpointB.GetProposedUpgrade().Fields.Version,
						types.AttributeKeyUpgradeOrdering:       path.EndpointB.GetProposedUpgrade().Fields.Ordering.String(),
						types.AttributeKeyUpgradeSequence:       fmt.Sprintf("%d", path.EndpointB.GetChannel().UpgradeSequence),
					},
					sdk.EventTypeMessage: {
						sdk.AttributeKeyModule: types.AttributeValueCategory,
					},
				}
				ibctesting.AssertEventsLegacy(&suite.Suite, expEvents, events)
			}
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
		expResult func(res *channeltypes.MsgChannelUpgradeAckResponse, err error)
		expEvents bool
	}{
		{
			"success, no pending in-flight packets",
			func() {},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(channeltypes.FLUSHCOMPLETE, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)
			},
			true,
		},
		{
			"success, pending in-flight packets",
			func() {
				portID := path.EndpointA.ChannelConfig.PortID
				channelID := path.EndpointA.ChannelID
				// Set a dummy packet commitment to simulate in-flight packets
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), portID, channelID, 1, []byte("hash"))
			},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(channeltypes.FLUSHING, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)
			},
			true,
		},
		{
			"module capability not found",
			func() {
				msg.PortId = ibctesting.InvalidID
				msg.ChannelId = ibctesting.InvalidID
			},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)

				suite.Require().ErrorIs(err, capabilitytypes.ErrCapabilityNotFound)
			},
			false,
		},
		{
			"core handler returns error and no upgrade error receipt is written",
			func() {
				// force an error by overriding the channel state to an invalid value
				channel := path.EndpointA.GetChannel()
				channel.State = channeltypes.CLOSED

				path.EndpointA.SetChannel(channel)
			},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)
				suite.Require().ErrorIs(err, channeltypes.ErrInvalidChannelState)

				errorReceipt, found := suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().Empty(errorReceipt)
				suite.Require().False(found)
			},
			false,
		},
		{
			"core handler returns error and writes upgrade error receipt",
			func() {
				// force an upgrade error by modifying the channel upgrade ordering to an incompatible value
				upgrade := path.EndpointA.GetChannelUpgrade()
				upgrade.Fields.Ordering = channeltypes.NONE

				path.EndpointA.SetChannelUpgrade(upgrade)
			},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, err error) {
				suite.Require().NoError(err)

				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.FAILURE, res.Result)

				errorReceipt, found := suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found)
				suite.Require().Equal(uint64(1), errorReceipt.Sequence)
			},
			false,
		},
		{
			"application callback returns error and error receipt is written",
			func() {
				suite.chainA.GetSimApp().IBCMockModule.IBCApp.OnChanUpgradeAck = func(
					ctx sdk.Context, portID, channelID, counterpartyVersion string,
				) error {
					// set arbitrary value in store to mock application state changes
					store := ctx.KVStore(suite.chainA.GetSimApp().GetKey(exported.ModuleName))
					store.Set([]byte("foo"), []byte("bar"))
					return fmt.Errorf("mock app callback failed")
				}
			},
			func(res *channeltypes.MsgChannelUpgradeAckResponse, err error) {
				suite.Require().NoError(err)

				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.FAILURE, res.Result)

				errorReceipt, found := suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found)
				suite.Require().Equal(uint64(1), errorReceipt.Sequence)

				// assert application state changes are not committed
				store := suite.chainA.GetContext().KVStore(suite.chainA.GetSimApp().GetKey(exported.ModuleName))
				suite.Require().False(store.Has([]byte("foo")))
			},
			false,
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

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

			proofChannel, proofUpgrade, proofHeight := path.EndpointB.QueryChannelUpgradeProof()

			msg = &channeltypes.MsgChannelUpgradeAck{
				PortId:              path.EndpointA.ChannelConfig.PortID,
				ChannelId:           path.EndpointA.ChannelID,
				CounterpartyUpgrade: counterpartyUpgrade,
				ProofChannel:        proofChannel,
				ProofUpgrade:        proofUpgrade,
				ProofHeight:         proofHeight,
				Signer:              suite.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			res, err := suite.chainA.GetSimApp().GetIBCKeeper().ChannelUpgradeAck(suite.chainA.GetContext(), msg)

			tc.expResult(res, err)

			// events := ctx.EventManager().Events().ToABCIEvents()
			// expEvents := ibctesting.EventsMap{
			//	types.EventTypeChannelUpgradeAck: {
			//		types.AttributeKeyPortID:                path.EndpointA.ChannelConfig.PortID,
			//		types.AttributeKeyChannelID:             path.EndpointA.ChannelID,
			//		types.AttributeCounterpartyPortID:       path.EndpointB.ChannelConfig.PortID,
			//		types.AttributeCounterpartyChannelID:    path.EndpointB.ChannelID,
			//		types.AttributeKeyUpgradeConnectionHops: upgrade.Fields.ConnectionHops[0],
			//		types.AttributeKeyUpgradeVersion:        upgrade.Fields.Version,
			//		types.AttributeKeyUpgradeOrdering:       upgrade.Fields.Ordering.String(),
			//		types.AttributeKeyUpgradeSequence:       fmt.Sprintf("%d", channel.UpgradeSequence),
			//	},
			//	sdk.EventTypeMessage: {
			//		sdk.AttributeKeyModule: types.AttributeValueCategory,
			//	},
			// }
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
		expResult func(res *channeltypes.MsgChannelUpgradeConfirmResponse, err error)
	}{
		{
			"success, no pending in-flight packets",
			func() {},
			func(res *channeltypes.MsgChannelUpgradeConfirmResponse, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)

				channel := path.EndpointB.GetChannel()
				suite.Require().Equal(channeltypes.OPEN, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)
			},
		},
		{
			"success, pending in-flight packets on init chain",
			func() {
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				suite.coordinator.Setup(path)

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

				proofChannel, proofUpgrade, proofHeight := path.EndpointA.QueryChannelUpgradeProof()

				msg = &channeltypes.MsgChannelUpgradeConfirm{
					PortId:                   path.EndpointB.ChannelConfig.PortID,
					ChannelId:                path.EndpointB.ChannelID,
					CounterpartyChannelState: counterpartyChannelState,
					CounterpartyUpgrade:      counterpartyUpgrade,
					ProofChannel:             proofChannel,
					ProofUpgrade:             proofUpgrade,
					ProofHeight:              proofHeight,
					Signer:                   suite.chainA.SenderAccount.GetAddress().String(),
				}
			},
			func(res *channeltypes.MsgChannelUpgradeConfirmResponse, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(channeltypes.FLUSHING, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)

				channel = path.EndpointB.GetChannel()
				suite.Require().Equal(channeltypes.FLUSHCOMPLETE, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)
			},
		},
		{
			"success, pending in-flight packets on try chain",
			func() {
				portID, channelID := path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID
				suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.SetPacketCommitment(suite.chainB.GetContext(), portID, channelID, 1, []byte("hash"))
			},
			func(res *channeltypes.MsgChannelUpgradeConfirmResponse, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)

				channel := path.EndpointB.GetChannel()
				suite.Require().Equal(channeltypes.FLUSHING, channel.State)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)
			},
		},
		{
			"module capability not found",
			func() {
				msg.PortId = ibctesting.InvalidID
				msg.ChannelId = ibctesting.InvalidID
			},
			func(res *channeltypes.MsgChannelUpgradeConfirmResponse, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)

				suite.Require().ErrorIs(err, capabilitytypes.ErrCapabilityNotFound)
			},
		},
		{
			"core handler returns error and no upgrade error receipt is written",
			func() {
				// force an error by overriding the channel state to an invalid value
				channel := path.EndpointB.GetChannel()
				channel.State = channeltypes.CLOSED

				path.EndpointB.SetChannel(channel)
			},
			func(res *channeltypes.MsgChannelUpgradeConfirmResponse, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)
				suite.Require().ErrorIs(err, channeltypes.ErrInvalidChannelState)

				errorReceipt, found := suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				suite.Require().Empty(errorReceipt)
				suite.Require().False(found)
			},
		},
		{
			"core handler returns error and writes upgrade error receipt",
			func() {
				// force an upgrade error by modifying the counterparty channel upgrade timeout to be no longer valid
				upgrade := path.EndpointA.GetChannelUpgrade()
				upgrade.Timeout = channeltypes.NewTimeout(clienttypes.ZeroHeight(), 0)

				path.EndpointA.SetChannelUpgrade(upgrade)

				suite.coordinator.CommitBlock(suite.chainA)

				err := path.EndpointB.UpdateClient()
				suite.Require().NoError(err)

				proofChannel, proofUpgrade, proofHeight := path.EndpointA.QueryChannelUpgradeProof()

				msg.CounterpartyUpgrade = upgrade
				msg.ProofChannel = proofChannel
				msg.ProofUpgrade = proofUpgrade
				msg.ProofHeight = proofHeight
			},
			func(res *channeltypes.MsgChannelUpgradeConfirmResponse, err error) {
				suite.Require().NoError(err)

				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.FAILURE, res.Result)

				errorReceipt, found := suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				suite.Require().True(found)
				suite.Require().Equal(uint64(1), errorReceipt.Sequence)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

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

			proofChannel, proofUpgrade, proofHeight := path.EndpointA.QueryChannelUpgradeProof()

			msg = &channeltypes.MsgChannelUpgradeConfirm{
				PortId:                   path.EndpointB.ChannelConfig.PortID,
				ChannelId:                path.EndpointB.ChannelID,
				CounterpartyChannelState: counterpartyChannelState,
				CounterpartyUpgrade:      counterpartyUpgrade,
				ProofChannel:             proofChannel,
				ProofUpgrade:             proofUpgrade,
				ProofHeight:              proofHeight,
				Signer:                   suite.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			res, err := suite.chainB.GetSimApp().GetIBCKeeper().ChannelUpgradeConfirm(suite.chainB.GetContext(), msg)

			tc.expResult(res, err)

			// events := ctx.EventManager().Events().ToABCIEvents()
			// expEvents := ibctesting.EventsMap{
			//	types.EventTypeChannelUpgradeOpen: {
			//		types.AttributeKeyPortID:                path.EndpointA.ChannelConfig.PortID,
			//		types.AttributeKeyChannelID:             path.EndpointA.ChannelID,
			//		types.AttributeCounterpartyPortID:       path.EndpointB.ChannelConfig.PortID,
			//		types.AttributeCounterpartyChannelID:    path.EndpointB.ChannelID,
			//		types.AttributeKeyChannelState:          types.OPEN.String(),
			//		types.AttributeKeyUpgradeConnectionHops: channel.ConnectionHops[0],
			//		types.AttributeKeyUpgradeVersion:        channel.Version,
			//		types.AttributeKeyUpgradeOrdering:       channel.Ordering.String(),
			//		types.AttributeKeyUpgradeSequence:       fmt.Sprintf("%d", channel.UpgradeSequence),
			//	},
			//	sdk.EventTypeMessage: {
			//		sdk.AttributeKeyModule: types.AttributeValueCategory,
			//	},
			// }
			// ibctesting.AssertEventsLegacy(&suite.Suite, expEvents, events)
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
		expResult func(res *channeltypes.MsgChannelUpgradeOpenResponse, err error)
	}{
		{
			"success",
			func() {},
			func(res *channeltypes.MsgChannelUpgradeOpenResponse, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(channeltypes.OPEN, channel.State)
			},
		},
		{
			"module capability not found",
			func() {
				msg.PortId = ibctesting.InvalidID
				msg.ChannelId = ibctesting.InvalidID
			},
			func(res *channeltypes.MsgChannelUpgradeOpenResponse, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)

				suite.Require().ErrorIs(err, capabilitytypes.ErrCapabilityNotFound)
			},
		},
		{
			"core handler fails",
			func() {
				channel := path.EndpointA.GetChannel()
				channel.State = channeltypes.FLUSHING
				path.EndpointA.SetChannel(channel)
			},
			func(res *channeltypes.MsgChannelUpgradeOpenResponse, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)

				suite.Require().ErrorIs(err, channeltypes.ErrInvalidChannelState)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

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
			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofChannel, proofHeight := path.EndpointB.QueryProof(channelKey)

			msg = &channeltypes.MsgChannelUpgradeOpen{
				PortId:                   path.EndpointA.ChannelConfig.PortID,
				ChannelId:                path.EndpointA.ChannelID,
				CounterpartyChannelState: counterpartyChannel.State,
				ProofChannel:             proofChannel,
				ProofHeight:              proofHeight,
				Signer:                   suite.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			res, err := suite.chainA.GetSimApp().GetIBCKeeper().ChannelUpgradeOpen(suite.chainA.GetContext(), msg)

			tc.expResult(res, err)
			// events := ctx.EventManager().Events().ToABCIEvents()
			// expEvents := ibctesting.EventsMap{
			//	types.EventTypeChannelUpgradeOpen: {
			//		types.AttributeKeyPortID:                path.EndpointA.ChannelConfig.PortID,
			//		types.AttributeKeyChannelID:             path.EndpointA.ChannelID,
			//		types.AttributeCounterpartyPortID:       path.EndpointB.ChannelConfig.PortID,
			//		types.AttributeCounterpartyChannelID:    path.EndpointB.ChannelID,
			//		types.AttributeKeyChannelState:          types.OPEN.String(),
			//		types.AttributeKeyUpgradeConnectionHops: channel.ConnectionHops[0],
			//		types.AttributeKeyUpgradeVersion:        channel.Version,
			//		types.AttributeKeyUpgradeOrdering:       channel.Ordering.String(),
			//		types.AttributeKeyUpgradeSequence:       fmt.Sprintf("%d", channel.UpgradeSequence),
			//	},
			//	sdk.EventTypeMessage: {
			//		sdk.AttributeKeyModule: types.AttributeValueCategory,
			//	},
			// }
			// ibctesting.AssertEventsLegacy(&suite.Suite, expEvents, events)
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
		expResult func(res *channeltypes.MsgChannelUpgradeCancelResponse, err error)
	}{
		{
			"success",
			func() {},
			func(res *channeltypes.MsgChannelUpgradeCancelResponse, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				channel := path.EndpointA.GetChannel()
				// Channel state should be reverted back to open.
				suite.Require().Equal(channeltypes.OPEN, channel.State)
				// Upgrade sequence should be changed to match sequence on error receipt.
				suite.Require().Equal(uint64(2), channel.UpgradeSequence)
			},
		},
		{
			"capability not found",
			func() {
				msg.ChannelId = ibctesting.InvalidID
			},
			func(res *channeltypes.MsgChannelUpgradeCancelResponse, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)

				channel := path.EndpointA.GetChannel()
				suite.Require().ErrorIs(err, capabilitytypes.ErrCapabilityNotFound)
				suite.Require().Equal(channeltypes.OPEN, channel.State)
				// Upgrade sequence should not be changed.
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)
			},
		},
		{
			"core handler fails: invalid proof",
			func() {
				msg.ProofErrorReceipt = []byte("invalid proof")
				// Force set to STATE_FLUSHCOMPLETE to check that state is not changed.
				suite.Require().NoError(path.EndpointA.SetChannelState(channeltypes.FLUSHCOMPLETE))
			},
			func(res *channeltypes.MsgChannelUpgradeCancelResponse, err error) {
				suite.Require().Error(err)
				suite.Require().Nil(res)

				channel := path.EndpointA.GetChannel()
				suite.Require().ErrorIs(err, commitmenttypes.ErrInvalidProof)
				// Channel state should not be changed.
				suite.Require().Equal(channeltypes.FLUSHCOMPLETE, channel.State)
				// Upgrade sequence should not be changed.
				suite.Require().Equal(uint64(1), channel.UpgradeSequence)
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			// configure the channel upgrade version on testing endpoints
			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			// cause the upgrade to fail on chain b so an error receipt is written.
			// if the counterparty (chain A) upgrade sequence is less than the current sequence, (chain B)
			// an upgrade error will be returned by chain B during ChanUpgradeTry.
			channel := path.EndpointA.GetChannel()
			channel.UpgradeSequence = 1
			path.EndpointA.SetChannel(channel)

			channel = path.EndpointB.GetChannel()
			channel.UpgradeSequence = 2
			path.EndpointB.SetChannel(channel)

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

			res, err := suite.chainA.GetSimApp().GetIBCKeeper().ChannelUpgradeCancel(suite.chainA.GetContext(), msg)
			tc.expResult(res, err)

			// we need to find the event values from the proposed upgrade as the actual upgrade has been deleted.
			// proposedUpgrade := path.EndpointA.GetProposedUpgrade()
			// events := ctx.EventManager().Events().ToABCIEvents()
			// expEvents := ibctesting.EventsMap{
			//	types.EventTypeChannelUpgradeCancel: {
			//		types.AttributeKeyPortID:                path.EndpointA.ChannelConfig.PortID,
			//		types.AttributeKeyChannelID:             path.EndpointA.ChannelID,
			//		types.AttributeCounterpartyPortID:       path.EndpointB.ChannelConfig.PortID,
			//		types.AttributeCounterpartyChannelID:    path.EndpointB.ChannelID,
			//		types.AttributeKeyUpgradeConnectionHops: proposedUpgrade.Fields.ConnectionHops[0],
			//		types.AttributeKeyUpgradeVersion:        proposedUpgrade.Fields.Version,
			//		types.AttributeKeyUpgradeOrdering:       proposedUpgrade.Fields.Ordering.String(),
			//		types.AttributeKeyUpgradeSequence:       fmt.Sprintf("%d", channel.UpgradeSequence),
			//	},
			//	sdk.EventTypeMessage: {
			//		sdk.AttributeKeyModule: types.AttributeValueCategory,
			//	},
			// }
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
		expResult func(res *channeltypes.MsgChannelUpgradeTimeoutResponse, err error)
	}{
		{
			"success",
			func() {
				timeoutUpgrade()

				suite.Require().NoError(path.EndpointA.UpdateClient())

				channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				channelProof, proofHeight := path.EndpointB.QueryProof(channelKey)

				msg.ProofChannel = channelProof
				msg.ProofHeight = proofHeight
			},
			func(res *channeltypes.MsgChannelUpgradeTimeoutResponse, err error) {
				suite.Require().NoError(err)
				channel := path.EndpointA.GetChannel()

				suite.Require().Equalf(channeltypes.OPEN, channel.State, "channel state should be %s", channeltypes.OPEN.String())

				_, found := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().False(found, "channel upgrade should be nil")

				suite.Require().NotNil(res)
				suite.Require().Equal(channeltypes.SUCCESS, res.Result)
			},
		},
		{
			"capability not found",
			func() {
				msg.ChannelId = ibctesting.InvalidID
			},
			func(res *channeltypes.MsgChannelUpgradeTimeoutResponse, err error) {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, capabilitytypes.ErrCapabilityNotFound)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equalf(channeltypes.FLUSHCOMPLETE, channel.State, "channel state should be %s", channeltypes.OPEN)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence, "channel upgrade sequence should not incremented")

				_, found := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found, "channel upgrade should not be nil")
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
			func(res *channeltypes.MsgChannelUpgradeTimeoutResponse, err error) {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, commitmenttypes.ErrInvalidProof)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equalf(channeltypes.FLUSHCOMPLETE, channel.State, "channel state should be %s", channeltypes.OPEN)
				suite.Require().Equal(uint64(1), channel.UpgradeSequence, "channel upgrade sequence should not incremented")

				_, found := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found, "channel upgrade should not be nil")
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

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

			tc.expResult(res, err)
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
			suite.coordinator.SetupClients(path)
			validAuthority := suite.chainA.App.GetIBCKeeper().GetAuthority()
			plan := upgradetypes.Plan{
				Name:   "upgrade IBC clients",
				Height: 1000,
			}
			// update trusting period
			clientState := path.EndpointB.GetClientState()
			clientState.(*ibctm.ClientState).TrustingPeriod += 100

			var err error
			msg, err = clienttypes.NewMsgIBCSoftwareUpgrade(
				validAuthority,
				plan,
				clientState,
			)

			suite.Require().NoError(err)

			tc.malleate()

			_, err = keeper.Keeper.IBCSoftwareUpgrade(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), msg)

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
				suite.Require().True(errors.Is(err, tc.expError))
			}
		})
	}
}

// TestUpdateClientParams tests the UpdateClientParams rpc handler
func (suite *KeeperTestSuite) TestUpdateClientParams() {
	signer := suite.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name    string
		msg     *clienttypes.MsgUpdateParams
		expPass bool
	}{
		{
			"success: valid signer and default params",
			clienttypes.NewMsgUpdateParams(signer, clienttypes.DefaultParams()),
			true,
		},
		{
			"failure: malformed signer address",
			clienttypes.NewMsgUpdateParams(ibctesting.InvalidID, clienttypes.DefaultParams()),
			false,
		},
		{
			"failure: empty signer address",
			clienttypes.NewMsgUpdateParams("", clienttypes.DefaultParams()),
			false,
		},
		{
			"failure: whitespace signer address",
			clienttypes.NewMsgUpdateParams("    ", clienttypes.DefaultParams()),
			false,
		},
		{
			"failure: unauthorized signer address",
			clienttypes.NewMsgUpdateParams(ibctesting.TestAccAddress, clienttypes.DefaultParams()),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			_, err := keeper.Keeper.UpdateClientParams(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), tc.msg)
			if tc.expPass {
				suite.Require().NoError(err)
				p := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(suite.chainA.GetContext())
				suite.Require().Equal(tc.msg.Params, p)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// TestUpdateConnectionParams tests the UpdateConnectionParams rpc handler
func (suite *KeeperTestSuite) TestUpdateConnectionParams() {
	signer := suite.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name    string
		msg     *connectiontypes.MsgUpdateParams
		expPass bool
	}{
		{
			"success: valid signer and default params",
			connectiontypes.NewMsgUpdateParams(signer, connectiontypes.DefaultParams()),
			true,
		},
		{
			"failure: malformed signer address",
			connectiontypes.NewMsgUpdateParams(ibctesting.InvalidID, connectiontypes.DefaultParams()),
			false,
		},
		{
			"failure: empty signer address",
			connectiontypes.NewMsgUpdateParams("", connectiontypes.DefaultParams()),
			false,
		},
		{
			"failure: whitespace signer address",
			connectiontypes.NewMsgUpdateParams("    ", connectiontypes.DefaultParams()),
			false,
		},
		{
			"failure: unauthorized signer address",
			connectiontypes.NewMsgUpdateParams(ibctesting.TestAccAddress, connectiontypes.DefaultParams()),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			_, err := keeper.Keeper.UpdateConnectionParams(*suite.chainA.App.GetIBCKeeper(), suite.chainA.GetContext(), tc.msg)
			if tc.expPass {
				suite.Require().NoError(err)
				p := suite.chainA.App.GetIBCKeeper().ConnectionKeeper.GetParams(suite.chainA.GetContext())
				suite.Require().Equal(tc.msg.Params, p)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
