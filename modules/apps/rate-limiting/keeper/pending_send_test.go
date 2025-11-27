package keeper_test

import "fmt"

func (s *KeeperTestSuite) TestPendingSendPacketPrefix() {
	// Store 5 packets across two channels
	sendPackets := []string{}
	for _, channelID := range []string{"channel-0", "channel-1"} {
		for sequence := range uint64(5) {
			s.chainA.GetSimApp().RateLimitKeeper.SetPendingSendPacket(s.chainA.GetContext(), channelID, sequence)
			sendPackets = append(sendPackets, fmt.Sprintf("%s/%d", channelID, sequence))
		}
	}

	// Check that they each sequence number is found
	for _, channelID := range []string{"channel-0", "channel-1"} {
		for sequence := range uint64(5) {
			found := s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(s.chainA.GetContext(), channelID, sequence)
			s.Require().True(found, "send packet should have been found - channel %s, sequence: %d", channelID, sequence)
		}
	}

	// Check lookup of all sequence numbers
	actualSendPackets := s.chainA.GetSimApp().RateLimitKeeper.GetAllPendingSendPackets(s.chainA.GetContext())
	s.Require().Equal(sendPackets, actualSendPackets, "all send packets")

	// Remove 0 sequence numbers and all sequence numbers from channel-0
	s.chainA.GetSimApp().RateLimitKeeper.RemovePendingSendPacket(s.chainA.GetContext(), "channel-0", 0)
	s.chainA.GetSimApp().RateLimitKeeper.RemovePendingSendPacket(s.chainA.GetContext(), "channel-1", 0)
	s.chainA.GetSimApp().RateLimitKeeper.RemoveAllChannelPendingSendPackets(s.chainA.GetContext(), "channel-0")

	// Check that only the remaining sequences are found
	for _, channelID := range []string{"channel-0", "channel-1"} {
		for sequence := range uint64(5) {
			expected := (channelID == "channel-1") && (sequence != 0)
			actual := s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(s.chainA.GetContext(), channelID, sequence)
			s.Require().Equal(expected, actual, "send packet after removal - channel: %s, sequence: %d", channelID, sequence)
		}
	}
}
