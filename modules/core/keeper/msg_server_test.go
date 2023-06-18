package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/cosmos/ibc-go/v7/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
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
func (s *KeeperTestSuite) TestHandleRecvPacket() {
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
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, false},
		{"success: UNORDERED", func() {
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, false},
		{"success: UNORDERED out of order packet", func() {
			// setup uses an UNORDERED channel
			s.coordinator.Setup(path)

			// attempts to receive packet with sequence 10 without receiving packet with sequence 1
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}
		}, true, false},
		{"success: OnRecvPacket callback returns revert=true", func() {
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockFailPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockFailPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, true},
		{"success: ORDERED - async acknowledgement", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)
			async = true

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibcmock.MockAsyncPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibcmock.MockAsyncPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, false},
		{"success: UNORDERED - async acknowledgement", func() {
			s.coordinator.Setup(path)
			async = true

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibcmock.MockAsyncPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibcmock.MockAsyncPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, true, false},
		{"failure: ORDERED out of order packet", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			// attempts to receive packet with sequence 10 without receiving packet with sequence 1
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}
		}, false, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			s.Require().NotNil(packet)
		}, false, false},
		{"packet not sent", func() {
			s.coordinator.Setup(path)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, false, false},
		{"successful no-op: ORDERED - packet already received (replay)", func() {
			// mock will panic if application callback is called twice on the same packet
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)
		}, true, false},
		{"successful no-op: UNORDERED - packet already received (replay)", func() {
			// mock will panic if application callback is called twice on the same packet
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)
		}, true, false},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset
			async = false // reset
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

			_, err := keeper.Keeper.RecvPacket(*s.chainB.App.GetIBCKeeper(), sdk.WrapSDKContext(s.chainB.GetContext()), msg)

			if tc.expPass {
				s.Require().NoError(err)

				// replay should not fail since it will be treated as a no-op
				_, err := keeper.Keeper.RecvPacket(*s.chainB.App.GetIBCKeeper(), sdk.WrapSDKContext(s.chainB.GetContext()), msg)
				s.Require().NoError(err)

				// check that callback state was handled correctly
				_, exists := s.chainB.GetSimApp().ScopedIBCMockKeeper.GetCapability(s.chainB.GetContext(), ibcmock.GetMockRecvCanaryCapabilityName(packet))
				if tc.expRevert {
					s.Require().False(exists, "capability exists in store even after callback reverted")
				} else {
					s.Require().True(exists, "callback state not persisted when revert is false")
				}

				// verify if ack was written
				ack, found := s.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(s.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

				if async {
					s.Require().Nil(ack)
					s.Require().False(found)

				} else {
					s.Require().NotNil(ack)
					s.Require().True(found)
				}
			} else {
				s.Require().Error(err)
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
		expPass  bool
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)
		}, true},
		{"success: UNORDERED", func() {
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)
		}, true},
		{"success: UNORDERED acknowledge out of order packet", func() {
			// setup uses an UNORDERED channel
			s.coordinator.Setup(path)

			// attempts to acknowledge ack with sequence 10 without acknowledging ack with sequence 1 (removing packet commitment)
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)
			}
		}, true},
		{"failure: ORDERED acknowledge out of order packet", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			// attempts to acknowledge ack with sequence 10 without acknowledging ack with sequence 1 (removing packet commitment
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)
			}
		}, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			s.Require().NotNil(packet)
		}, false},
		{"packet not received", func() {
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, false},
		{"successful no-op: ORDERED - packet already acknowledged (replay)", func() {
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
			s.Require().NoError(err)
		}, true},
		{"successful no-op: UNORDERED - packet already acknowledged (replay)", func() {
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
			s.Require().NoError(err)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc

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

			_, err := keeper.Keeper.Acknowledgement(*s.chainA.App.GetIBCKeeper(), sdk.WrapSDKContext(s.chainA.GetContext()), msg)

			if tc.expPass {
				s.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := s.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(s.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				s.Require().False(has)

				// replay should not error as it is treated as a no-op
				_, err := keeper.Keeper.Acknowledgement(*s.chainA.App.GetIBCKeeper(), sdk.WrapSDKContext(s.chainA.GetContext()), msg)
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
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
		expPass  bool
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

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
		}, true},
		{"success: UNORDERED", func() {
			s.coordinator.Setup(path)

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
		}, true},
		{"success: UNORDERED timeout out of order packet", func() {
			// setup uses an UNORDERED channel
			s.coordinator.Setup(path)

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
		}, true},
		{"success: ORDERED timeout out of order packet", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

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
		}, true},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			s.Require().NotNil(packet)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, false},
		{"successful no-op: UNORDERED - packet not sent", func() {
			s.coordinator.Setup(path)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 1), 0)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, true},
	}

	for _, tc := range testCases {
		tc := tc

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

			_, err := keeper.Keeper.Timeout(*s.chainA.App.GetIBCKeeper(), sdk.WrapSDKContext(s.chainA.GetContext()), msg)

			if tc.expPass {
				s.Require().NoError(err)

				// replay should not return an error as it is treated as a no-op
				_, err := keeper.Keeper.Timeout(*s.chainA.App.GetIBCKeeper(), sdk.WrapSDKContext(s.chainA.GetContext()), msg)
				s.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := s.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(s.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				s.Require().False(has)

			} else {
				s.Require().Error(err)
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
		expPass  bool
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())

			// close counterparty channel
			err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
			s.Require().NoError(err)
		}, true},
		{"success: UNORDERED", func() {
			s.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
			s.Require().NoError(err)
		}, true},
		{"success: UNORDERED timeout out of order packet", func() {
			// setup uses an UNORDERED channel
			s.coordinator.Setup(path)

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
			err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
			s.Require().NoError(err)
		}, true},
		{"success: ORDERED timeout out of order packet", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

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
			err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
			s.Require().NoError(err)
		}, true},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			s.Require().NotNil(packet)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, false},
		{"successful no-op: UNORDERED - packet not sent", func() {
			s.coordinator.Setup(path)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 1), 0)
			packetKey = host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			err := path.EndpointB.SetChannelState(channeltypes.CLOSED)
			s.Require().NoError(err)
		}, true},
		{"ORDERED: channel not closed", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, false},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			proof, proofHeight := s.chainB.QueryProof(packetKey)

			channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			proofClosed, _ := s.chainB.QueryProof(channelKey)

			msg := channeltypes.NewMsgTimeoutOnClose(packet, 1, proof, proofClosed, proofHeight, s.chainA.SenderAccount.GetAddress().String())

			_, err := keeper.Keeper.TimeoutOnClose(*s.chainA.App.GetIBCKeeper(), sdk.WrapSDKContext(s.chainA.GetContext()), msg)

			if tc.expPass {
				s.Require().NoError(err)

				// replay should not return an error as it will be treated as a no-op
				_, err := keeper.Keeper.TimeoutOnClose(*s.chainA.App.GetIBCKeeper(), sdk.WrapSDKContext(s.chainA.GetContext()), msg)
				s.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := s.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(s.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				s.Require().False(has)

			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestUpgradeClient() {
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

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradeClient, _ := s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ := s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())

				msg, err = clienttypes.NewMsgUpgradeClient(path.EndpointA.ClientID, upgradedClient, upgradedConsState,
					proofUpgradeClient, proofUpgradedConsState, s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
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
			expPass: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		path = ibctesting.NewPath(s.chainA, s.chainB)
		s.coordinator.SetupClients(path)

		var err error
		clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
		revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

		newChainID, err = clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
		s.Require().NoError(err)

		newClientHeight = clienttypes.NewHeight(revisionNumber+1, clientState.GetLatestHeight().GetRevisionHeight()+1)

		tc.setup()

		_, err = keeper.Keeper.UpgradeClient(*s.chainA.App.GetIBCKeeper(), sdk.WrapSDKContext(s.chainA.GetContext()), msg)

		if tc.expPass {
			s.Require().NoError(err, "upgrade handler failed on valid case: %s", tc.name)
			newClient, ok := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().True(ok)
			newChainSpecifiedClient := newClient.ZeroCustomFields()
			s.Require().Equal(upgradedClient, newChainSpecifiedClient)
		} else {
			s.Require().Error(err, "upgrade handler passed on invalid case: %s", tc.name)
		}
	}
}

// TestUpdateClientParams tests the UpdateClientParams rpc handler
func (s *KeeperTestSuite) TestUpdateClientParams() {
	validAuthority := s.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name    string
		msg     *clienttypes.MsgUpdateParams
		expPass bool
	}{
		{
			"success: valid authority and default params",
			clienttypes.NewMsgUpdateParams(validAuthority, clienttypes.DefaultParams()),
			true,
		},
		{
			"failure: malformed authority address",
			clienttypes.NewMsgUpdateParams(ibctesting.InvalidID, clienttypes.DefaultParams()),
			false,
		},
		{
			"failure: empty authority address",
			clienttypes.NewMsgUpdateParams("", clienttypes.DefaultParams()),
			false,
		},
		{
			"failure: whitespace authority address",
			clienttypes.NewMsgUpdateParams("    ", clienttypes.DefaultParams()),
			false,
		},
		{
			"failure: unauthorized authority address",
			clienttypes.NewMsgUpdateParams(ibctesting.TestAccAddress, clienttypes.DefaultParams()),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()
			_, err := keeper.Keeper.UpdateClientParams(*s.chainA.App.GetIBCKeeper(), s.chainA.GetContext(), tc.msg)
			if tc.expPass {
				s.Require().NoError(err)
				p := s.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(s.chainA.GetContext())
				s.Require().Equal(tc.msg.Params, p)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// TestUpdateConnectionParams tests the UpdateConnectionParams rpc handler
func (s *KeeperTestSuite) TestUpdateConnectionParams() {
	validAuthority := s.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name    string
		msg     *connectiontypes.MsgUpdateParams
		expPass bool
	}{
		{
			"success: valid authority and default params",
			connectiontypes.NewMsgUpdateParams(validAuthority, connectiontypes.DefaultParams()),
			true,
		},
		{
			"failure: malformed authority address",
			connectiontypes.NewMsgUpdateParams(ibctesting.InvalidID, connectiontypes.DefaultParams()),
			false,
		},
		{
			"failure: empty authority address",
			connectiontypes.NewMsgUpdateParams("", connectiontypes.DefaultParams()),
			false,
		},
		{
			"failure: whitespace authority address",
			connectiontypes.NewMsgUpdateParams("    ", connectiontypes.DefaultParams()),
			false,
		},
		{
			"failure: unauthorized authority address",
			connectiontypes.NewMsgUpdateParams(ibctesting.TestAccAddress, connectiontypes.DefaultParams()),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()
			_, err := keeper.Keeper.UpdateConnectionParams(*s.chainA.App.GetIBCKeeper(), s.chainA.GetContext(), tc.msg)
			if tc.expPass {
				s.Require().NoError(err)
				p := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetParams(s.chainA.GetContext())
				s.Require().Equal(tc.msg.Params, p)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
