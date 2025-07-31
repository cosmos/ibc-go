package keeper_test

import (
	"reflect"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
)

// KeeperTestSuite is a testing suite to test keeper functions.
type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

// TestKeeperTestSuite runs all the tests within this package.
func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

// SetupTest creates a coordinator with 2 test chains.
func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	s.coordinator.CommitNBlocks(s.chainA, 2)
	s.coordinator.CommitNBlocks(s.chainB, 2)
	s.coordinator.CommitNBlocks(s.chainC, 2)
}

// TestSetChannel create clients and connections on both chains. It tests for the non-existence
// and existence of a channel in INIT on chainA.
func (s *KeeperTestSuite) TestSetChannel() {
	// create client and connections on both chains
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupConnections()

	// check for channel to be created on chainA
	found := s.chainA.App.GetIBCKeeper().ChannelKeeper.HasChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	s.False(found)

	path.SetChannelOrdered()

	// init channel
	err := path.EndpointA.ChanOpenInit()
	s.Require().NoError(err)

	storedChannel, found := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	// counterparty channel id is empty after open init
	expectedCounterparty := types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, "")

	s.True(found)
	s.Equal(types.INIT, storedChannel.State)
	s.Equal(types.ORDERED, storedChannel.Ordering)
	s.Equal(expectedCounterparty, storedChannel.Counterparty)
}

func (s *KeeperTestSuite) TestGetAppVersion() {
	// create client and connections on both chains
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupConnections()

	version, found := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetAppVersion(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	s.Require().False(found)
	s.Require().Empty(version)

	// init channel
	err := path.EndpointA.ChanOpenInit()
	s.Require().NoError(err)

	channelVersion, found := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetAppVersion(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	s.Require().True(found)
	s.Require().Equal(ibcmock.Version, channelVersion)
}

// TestGetAllChannelsWithPortPrefix verifies ports are filtered correctly using a port prefix.
func (s *KeeperTestSuite) TestGetAllChannelsWithPortPrefix() {
	const (
		secondChannelID        = "channel-1"
		differentChannelPortID = "different-portid"
	)

	allChannels := []types.IdentifiedChannel{
		types.NewIdentifiedChannel(transfertypes.PortID, ibctesting.FirstChannelID, types.Channel{}),
		types.NewIdentifiedChannel(differentChannelPortID, secondChannelID, types.Channel{}),
	}

	tests := []struct {
		name             string
		prefix           string
		allChannels      []types.IdentifiedChannel
		expectedChannels []types.IdentifiedChannel
	}{
		{
			name:             "transfer channel is retrieved with prefix",
			prefix:           "tra",
			allChannels:      allChannels,
			expectedChannels: []types.IdentifiedChannel{types.NewIdentifiedChannel(transfertypes.PortID, ibctesting.FirstChannelID, types.Channel{})},
		},
		{
			name:             "matches port with full name as prefix",
			prefix:           transfertypes.PortID,
			allChannels:      allChannels,
			expectedChannels: []types.IdentifiedChannel{types.NewIdentifiedChannel(transfertypes.PortID, ibctesting.FirstChannelID, types.Channel{})},
		},
		{
			name:             "no ports match prefix",
			prefix:           "wont-match-anything",
			allChannels:      allChannels,
			expectedChannels: nil,
		},
		{
			name:             "empty prefix matches everything",
			prefix:           "",
			allChannels:      allChannels,
			expectedChannels: allChannels,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()

			for _, ch := range tc.allChannels {
				s.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(s.chainA.GetContext(), ch.PortId, ch.ChannelId, types.Channel{})
			}

			ctxA := s.chainA.GetContext()

			actualChannels := s.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetAllChannelsWithPortPrefix(ctxA, tc.prefix)

			s.Require().True(containsAll(tc.expectedChannels, actualChannels))
		})
	}
}

// containsAll verifies if all elements in the expected slice exist in the actual slice
// independent of order.
func containsAll(expected, actual []types.IdentifiedChannel) bool {
	for _, expectedChannel := range expected {
		foundMatch := false
		for _, actualChannel := range actual {
			if reflect.DeepEqual(actualChannel, expectedChannel) {
				foundMatch = true
				break
			}
		}
		if !foundMatch {
			return false
		}
	}
	return true
}

// TestGetAllChannels creates multiple channels on chain A through various connections
// and tests their retrieval. 2 channels are on connA0 and 1 channel is on connA1
func (s *KeeperTestSuite) TestGetAllChannels() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.Setup()
	// channel0 on first connection on chainA
	counterparty0 := types.Counterparty{
		PortId:    path.EndpointB.ChannelConfig.PortID,
		ChannelId: path.EndpointB.ChannelID,
	}

	// path1 creates a second channel on first connection on chainA
	path1 := ibctesting.NewPath(s.chainA, s.chainB)
	path1.SetChannelOrdered()
	path1.EndpointA.ClientID = path.EndpointA.ClientID
	path1.EndpointB.ClientID = path.EndpointB.ClientID
	path1.EndpointA.ConnectionID = path.EndpointA.ConnectionID
	path1.EndpointB.ConnectionID = path.EndpointB.ConnectionID

	s.coordinator.CreateMockChannels(path1)
	counterparty1 := types.Counterparty{
		PortId:    path1.EndpointB.ChannelConfig.PortID,
		ChannelId: path1.EndpointB.ChannelID,
	}

	path2 := ibctesting.NewPath(s.chainA, s.chainB)
	path2.SetupConnections()

	// path2 creates a second channel on chainA
	err := path2.EndpointA.ChanOpenInit()
	s.Require().NoError(err)

	// counterparty channel id is empty after open init
	counterparty2 := types.Counterparty{
		PortId:    path2.EndpointB.ChannelConfig.PortID,
		ChannelId: "",
	}

	channel0 := types.NewChannel(
		types.OPEN, types.UNORDERED,
		counterparty0, []string{path.EndpointA.ConnectionID}, path.EndpointA.ChannelConfig.Version,
	)
	channel1 := types.NewChannel(
		types.OPEN, types.ORDERED,
		counterparty1, []string{path1.EndpointA.ConnectionID}, path1.EndpointA.ChannelConfig.Version,
	)
	channel2 := types.NewChannel(
		types.INIT, types.UNORDERED,
		counterparty2, []string{path2.EndpointA.ConnectionID}, path2.EndpointA.ChannelConfig.Version,
	)

	expChannels := []types.IdentifiedChannel{
		types.NewIdentifiedChannel(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel0),
		types.NewIdentifiedChannel(path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, channel1),
		types.NewIdentifiedChannel(path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID, channel2),
	}

	ctxA := s.chainA.GetContext()

	channels := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetAllChannels(ctxA)
	s.Require().Len(channels, len(expChannels))
	s.Require().Equal(expChannels, channels)
}

// TestGetAllSequences sets all packet sequences for two different channels on chain A and
// tests their retrieval.
func (s *KeeperTestSuite) TestGetAllSequences() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.Setup()

	path1 := ibctesting.NewPath(s.chainA, s.chainB)
	path1.SetChannelOrdered()
	path1.EndpointA.ClientID = path.EndpointA.ClientID
	path1.EndpointB.ClientID = path.EndpointB.ClientID
	path1.EndpointA.ConnectionID = path.EndpointA.ConnectionID
	path1.EndpointB.ConnectionID = path.EndpointB.ConnectionID

	s.coordinator.CreateMockChannels(path1)

	seq1 := types.NewPacketSequence(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1)
	seq2 := types.NewPacketSequence(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 2)
	seq3 := types.NewPacketSequence(path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, 3)

	// seq1 should be overwritten by seq2
	expSeqs := []types.PacketSequence{seq2, seq3}

	ctxA := s.chainA.GetContext()

	for _, seq := range []types.PacketSequence{seq1, seq2, seq3} {
		s.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceSend(ctxA, seq.PortId, seq.ChannelId, seq.Sequence)
		s.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceRecv(ctxA, seq.PortId, seq.ChannelId, seq.Sequence)
		s.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceAck(ctxA, seq.PortId, seq.ChannelId, seq.Sequence)
	}

	sendSeqs := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetAllPacketSendSeqs(ctxA)
	recvSeqs := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetAllPacketRecvSeqs(ctxA)
	ackSeqs := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetAllPacketAckSeqs(ctxA)
	s.Len(sendSeqs, 2)
	s.Len(recvSeqs, 2)
	s.Len(ackSeqs, 2)

	s.Equal(expSeqs, sendSeqs)
	s.Equal(expSeqs, recvSeqs)
	s.Equal(expSeqs, ackSeqs)
}

// TestGetAllPacketState creates a set of acks, packet commitments, and receipts on two different
// channels on chain A and tests their retrieval.
func (s *KeeperTestSuite) TestGetAllPacketState() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.Setup()

	path1 := ibctesting.NewPath(s.chainA, s.chainB)
	path1.EndpointA.ClientID = path.EndpointA.ClientID
	path1.EndpointB.ClientID = path.EndpointB.ClientID
	path1.EndpointA.ConnectionID = path.EndpointA.ConnectionID
	path1.EndpointB.ConnectionID = path.EndpointB.ConnectionID

	s.coordinator.CreateMockChannels(path1)

	// channel 0 acks
	ack1 := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, []byte("ack"))
	ack2 := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 2, []byte("ack"))

	// duplicate ack
	ack2dup := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 2, []byte("ack"))

	// channel 1 acks
	ack3 := types.NewPacketState(path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, 1, []byte("ack"))

	// create channel 0 receipts
	receipt := string([]byte{byte(1)})
	rec1 := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, []byte(receipt))
	rec2 := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 2, []byte(receipt))

	// channel 1 receipts
	rec3 := types.NewPacketState(path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, 1, []byte(receipt))
	rec4 := types.NewPacketState(path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, 2, []byte(receipt))

	// channel 0 packet commitments
	comm1 := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, []byte("hash"))
	comm2 := types.NewPacketState(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 2, []byte("hash"))

	// channel 1 packet commitments
	comm3 := types.NewPacketState(path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, 1, []byte("hash"))
	comm4 := types.NewPacketState(path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, 2, []byte("hash"))

	expAcks := []types.PacketState{ack1, ack2, ack3}
	expReceipts := []types.PacketState{rec1, rec2, rec3, rec4}
	expCommitments := []types.PacketState{comm1, comm2, comm3, comm4}

	ctxA := s.chainA.GetContext()

	// set acknowledgements
	for _, ack := range []types.PacketState{ack1, ack2, ack2dup, ack3} {
		s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(ctxA, ack.PortId, ack.ChannelId, ack.Sequence, ack.Data)
	}

	// set packet receipts
	for _, rec := range expReceipts {
		s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(ctxA, rec.PortId, rec.ChannelId, rec.Sequence)
	}

	// set packet commitments
	for _, comm := range expCommitments {
		s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(ctxA, comm.PortId, comm.ChannelId, comm.Sequence, comm.Data)
	}

	acks := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetAllPacketAcks(ctxA)
	receipts := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetAllPacketReceipts(ctxA)
	commitments := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetAllPacketCommitments(ctxA)

	s.Require().Len(acks, len(expAcks))
	s.Require().Len(commitments, len(expCommitments))
	s.Require().Len(receipts, len(expReceipts))

	s.Require().Equal(expAcks, acks)
	s.Require().Equal(expReceipts, receipts)
	s.Require().Equal(expCommitments, commitments)
}

// TestSetSequence verifies that the keeper correctly sets the sequence counters.
func (s *KeeperTestSuite) TestSetSequence() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.Setup()

	ctxA := s.chainA.GetContext()
	one := uint64(1)

	// initialized channel has next send seq of 1
	seq, found := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceSend(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	s.True(found)
	s.Equal(one, seq)

	// initialized channel has next seq recv of 1
	seq, found = s.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	s.True(found)
	s.Equal(one, seq)

	// initialized channel has next seq ack of
	seq, found = s.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceAck(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	s.True(found)
	s.Equal(one, seq)

	nextSeqSend, nextSeqRecv, nextSeqAck := uint64(10), uint64(10), uint64(10)
	s.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceSend(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, nextSeqSend)
	s.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceRecv(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, nextSeqRecv)
	s.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceAck(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, nextSeqAck)

	storedNextSeqSend, found := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceSend(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	s.True(found)
	s.Equal(nextSeqSend, storedNextSeqSend)

	storedNextSeqRecv, found := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceSend(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	s.True(found)
	s.Equal(nextSeqRecv, storedNextSeqRecv)

	storedNextSeqAck, found := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceAck(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	s.True(found)
	s.Equal(nextSeqAck, storedNextSeqAck)
}

// TestGetAllPacketCommitmentsAtChannel verifies that the keeper returns all stored packet
// commitments for a specific channel. The test will store consecutive commitments up to the
// value of "seq" and then add non-consecutive up to the value of "maxSeq". A final commitment
// with the value maxSeq + 1 is set on a different channel.
func (s *KeeperTestSuite) TestGetAllPacketCommitmentsAtChannel() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.Setup()

	// create second channel
	path1 := ibctesting.NewPath(s.chainA, s.chainB)
	path1.SetChannelOrdered()
	path1.EndpointA.ClientID = path.EndpointA.ClientID
	path1.EndpointB.ClientID = path.EndpointB.ClientID
	path1.EndpointA.ConnectionID = path.EndpointA.ConnectionID
	path1.EndpointB.ConnectionID = path.EndpointB.ConnectionID

	s.coordinator.CreateMockChannels(path1)

	ctxA := s.chainA.GetContext()
	expectedSeqs := make(map[uint64]bool)
	hash := []byte("commitment")

	seq := uint64(15)
	maxSeq := uint64(25)
	s.Require().Greater(maxSeq, seq)

	// create consecutive commitments
	for i := uint64(1); i < seq; i++ {
		s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, i, hash)
		expectedSeqs[i] = true
	}

	// add non-consecutive commitments
	for i := seq; i < maxSeq; i += 2 {
		s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, i, hash)
		expectedSeqs[i] = true
	}

	// add sequence on different channel/port
	s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(ctxA, path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, maxSeq+1, hash)

	commitments := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetAllPacketCommitmentsAtChannel(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

	s.Len(commitments, len(expectedSeqs))
	// ensure above for loops occurred
	s.Require().NotEmpty(commitments)

	// verify that all the packet commitments were stored
	for _, packet := range commitments {
		s.True(expectedSeqs[packet.Sequence])
		s.Equal(path.EndpointA.ChannelConfig.PortID, packet.PortId)
		s.Equal(path.EndpointA.ChannelID, packet.ChannelId)
		s.Equal(hash, packet.Data)

		// prevent duplicates from passing checks
		expectedSeqs[packet.Sequence] = false
	}
}

// TestSetPacketAcknowledgement verifies that packet acknowledgements are correctly
// set in the keeper.
func (s *KeeperTestSuite) TestSetPacketAcknowledgement() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.Setup()

	ctxA := s.chainA.GetContext()
	seq := uint64(10)

	storedAckHash, found := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq)
	s.Require().False(found)
	s.Require().Nil(storedAckHash)

	ackHash := []byte("ackhash")
	s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq, ackHash)

	storedAckHash, found = s.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq)
	s.Require().True(found)
	s.Require().Equal(ackHash, storedAckHash)
	s.Require().True(s.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketAcknowledgement(ctxA, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq))
}

// TestGetV2Counterparty verifies that the v2 counterparty is correctly retrieved from v1 channel.
func (s *KeeperTestSuite) TestGetV2Counterparty() {
	var (
		path            *ibctesting.Path
		expCounterparty clientv2types.CounterpartyInfo
	)
	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			name:     "success",
			malleate: func() {},
		},
		{
			name: "channel not found",
			malleate: func() {
				path.EndpointA.ChannelID = "fake-channel"
				expCounterparty = clientv2types.CounterpartyInfo{}
			},
		},
		{
			name: "channel not OPEN",
			malleate: func() {
				channel, ok := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				s.Require().True(ok)
				channel.State = types.CLOSED
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
				expCounterparty = clientv2types.CounterpartyInfo{}
			},
		},
		{
			name: "channel not UNORDERED",
			malleate: func() {
				channel, ok := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				s.Require().True(ok)
				channel.Ordering = types.ORDERED
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
				expCounterparty = clientv2types.CounterpartyInfo{}
			},
		},
		{
			name: "connection not found",
			malleate: func() {
				channel, ok := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				s.Require().True(ok)
				channel.ConnectionHops = []string{"fake-connection"}
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
				expCounterparty = clientv2types.CounterpartyInfo{}
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.Setup()

			expCounterparty = clientv2types.CounterpartyInfo{
				ClientId:     path.EndpointB.ChannelID,
				MerklePrefix: [][]byte{[]byte("ibc"), []byte("")},
			}

			tc.malleate()

			counterparty, ok := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetV2Counterparty(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			s.Require().Equal(expCounterparty, counterparty)
			s.Require().Equal(ok, !reflect.DeepEqual(expCounterparty, clientv2types.CounterpartyInfo{}))
		})
	}
}
