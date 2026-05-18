package keeper_test

import "fmt"

func (s *KeeperTestSuite) TestPendingSendPacketPrefix() {
	// Store 5 packets across 4 channels
	channels := []string{"07-tendermint-1000", "07-tendermint-1005", "channel-1", "channel-11"}
	sendPackets := []string{}
	for _, channelID := range channels {
		for sequence := range uint64(5) {
			err := s.chainA.GetSimApp().RateLimitKeeper.SetPendingSendPacket(s.chainA.GetContext(), channelID, sequence)
			s.Require().NoError(err, "unexpected error setting pending send packet sequence - channel %s, sequence %s", channelID, sequence)
			sendPackets = append(sendPackets, fmt.Sprintf("%s/%d", channelID, sequence))
		}
	}

	// Check that each sequence number is found
	for _, channelID := range channels {
		for sequence := range uint64(5) {
			found, err := s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(s.chainA.GetContext(), channelID, sequence)
			s.Require().NoError(err, "unexpected error checking packet sent during current quota - channel %s, sequence %s", channelID, sequence)
			s.Require().True(found, "send packet should have been found - channel %s, sequence: %d", channelID, sequence)
		}
	}

	// Check lookup of all sequence numbers
	actualSendPackets, err := s.chainA.GetSimApp().RateLimitKeeper.GetAllPendingSendPackets(s.chainA.GetContext())
	s.Require().NoError(err, "unexpected error getting pending send packets")
	s.Require().Equal(sendPackets, actualSendPackets, "all send packets")

	// Remove 0 sequence numbers and all sequence numbers from channel-0 + 07-tendermint-1005
	for _, channelID := range channels {
		s.chainA.GetSimApp().RateLimitKeeper.RemovePendingSendPacket(s.chainA.GetContext(), channelID, 0)
	}
	err = s.chainA.GetSimApp().RateLimitKeeper.RemoveAllChannelPendingSendPackets(s.chainA.GetContext(), "channel-1")
	s.Require().NoError(err, "unexpected error removing all pending send packets - channel %s", "channel-1")
	err = s.chainA.GetSimApp().RateLimitKeeper.RemoveAllChannelPendingSendPackets(s.chainA.GetContext(), "07-tendermint-1005")
	s.Require().NoError(err, "unexpected error removing all pending send packets - channel %s", "07-tendermint-1005")

	// Check that only the remaining sequences are found
	for _, channelID := range channels {
		for sequence := range uint64(5) {
			removed := (channelID == "channel-1") || (channelID == "07-tendermint-1005") || (sequence == 0)
			actual, err := s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(s.chainA.GetContext(), channelID, sequence)
			s.Require().NoError(err, "unexpected error checking packet sent during current quota - channel %s, sequence %s", channelID, sequence)

			// Assert that if we did not remove the packet, then we
			// successfully find it when checking the quota
			s.Require().Equal(!removed, actual, "send packet after removal - channel: %s, sequence: %d", channelID, sequence)
		}
	}
}
