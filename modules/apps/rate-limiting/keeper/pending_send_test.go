package keeper_test

import "fmt"

func (s *KeeperTestSuite) TestPendingSendPacketPrefix() {
	// Store 5 packets across 4 channels
	channels := []string{"07-tendermint-1000", "07-tendermint-1005", "channel-1", "channel-11"}
	denoms := []string{"denom-a", "denom-b"}
	sendPackets := []string{}
	for _, channelID := range channels {
		for _, denom := range denoms {
			for sequence := range uint64(5) {
				err := s.chainA.GetSimApp().RateLimitKeeper.SetPendingSendPacket(s.chainA.GetContext(), channelID, sequence, denom)
				s.Require().NoError(err, "unexpected error setting pending send packet sequence - channel %s, sequence %s, denom %s", channelID, sequence, denom)
				sendPackets = append(sendPackets, fmt.Sprintf("%s/%d/%s", channelID, sequence, denom))
			}
		}
	}

	// Check that each sequence number is found
	for _, channelID := range channels {
		for _, denom := range denoms {
			for sequence := range uint64(5) {
				found, err := s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(s.chainA.GetContext(), channelID, sequence, denom)
				s.Require().NoError(err, "unexpected error checking packet sent during current quota - channel %s, sequence %s, denom %s", channelID, sequence, denom)
				s.Require().True(found, "send packet should have been found - channel %s, sequence: %d, denom: %s", channelID, sequence, denom)
			}
		}
	}

	// Check lookup of all sequence numbers
	actualSendPackets, err := s.chainA.GetSimApp().RateLimitKeeper.GetAllPendingSendPackets(s.chainA.GetContext())
	s.Require().NoError(err, "unexpected error getting pending send packets")
	s.Require().ElementsMatch(sendPackets, actualSendPackets, "all send packets")

	// Remove denom-a sequence 0 and all denom-scoped sequence numbers from channel-1 + 07-tendermint-1005
	for _, channelID := range channels {
		err = s.chainA.GetSimApp().RateLimitKeeper.RemovePendingSendPacket(s.chainA.GetContext(), channelID, 0, "denom-a")
		s.Require().NoError(err, "unexpected error removing pending send packet - channel %s, sequence 0", channelID)
	}
	err = s.chainA.GetSimApp().RateLimitKeeper.RemoveAllChannelPendingSendPackets(s.chainA.GetContext(), "channel-1", "denom-a")
	s.Require().NoError(err, "unexpected error removing all pending send packets - channel %s", "channel-1")
	err = s.chainA.GetSimApp().RateLimitKeeper.RemoveAllChannelPendingSendPackets(s.chainA.GetContext(), "07-tendermint-1005", "denom-b")
	s.Require().NoError(err, "unexpected error removing all pending send packets - channel %s", "07-tendermint-1005")

	// Check that only the remaining sequences are found
	for _, channelID := range channels {
		for _, denom := range denoms {
			for sequence := range uint64(5) {
				removed := (denom == "denom-a" && sequence == 0) || (channelID == "channel-1" && denom == "denom-a") || (channelID == "07-tendermint-1005" && denom == "denom-b")
				actual, err := s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(s.chainA.GetContext(), channelID, sequence, denom)
				s.Require().NoError(err, "unexpected error checking packet sent during current quota - channel %s, sequence %s, denom %s", channelID, sequence, denom)

				// Assert that if we did not remove the packet, then we
				// successfully find it when checking the quota
				s.Require().Equal(!removed, actual, "send packet after removal - channel: %s, sequence: %d, denom: %s", channelID, sequence, denom)
			}
		}
	}
}

func (s *KeeperTestSuite) TestPendingReceivePacketPrefix() {
	// Store 5 packets across 4 channels
	channels := []string{"07-tendermint-1000", "07-tendermint-1005", "channel-1", "channel-11"}
	denoms := []string{"denom-a", "denom-b"}
	for _, channelID := range channels {
		for _, denom := range denoms {
			for sequence := range uint64(5) {
				err := s.chainA.GetSimApp().RateLimitKeeper.SetPendingReceivePacket(s.chainA.GetContext(), channelID, sequence, denom)
				s.Require().NoError(err, "unexpected error setting pending receive packet sequence - channel %s, sequence %s, denom %s", channelID, sequence, denom)
			}
		}
	}

	// Check that each sequence number is found
	for _, channelID := range channels {
		for _, denom := range denoms {
			for sequence := range uint64(5) {
				found, err := s.chainA.GetSimApp().RateLimitKeeper.CheckPacketReceivedDuringCurrentQuota(s.chainA.GetContext(), channelID, sequence, denom)
				s.Require().NoError(err, "unexpected error checking packet received during current quota - channel %s, sequence %s, denom %s", channelID, sequence, denom)
				s.Require().True(found, "receive packet should have been found - channel %s, sequence: %d, denom: %s", channelID, sequence, denom)
			}
		}
	}

	// Remove denom-a sequence 0 and all denom-scoped sequence numbers from channel-1 + 07-tendermint-1005
	for _, channelID := range channels {
		err := s.chainA.GetSimApp().RateLimitKeeper.RemovePendingReceivePacket(s.chainA.GetContext(), channelID, 0, "denom-a")
		s.Require().NoError(err, "unexpected error removing pending receive packet - channel %s, sequence 0", channelID)
	}
	err := s.chainA.GetSimApp().RateLimitKeeper.RemoveAllChannelPendingReceivePackets(s.chainA.GetContext(), "channel-1", "denom-a")
	s.Require().NoError(err, "unexpected error removing all pending receive packets - channel %s", "channel-1")
	err = s.chainA.GetSimApp().RateLimitKeeper.RemoveAllChannelPendingReceivePackets(s.chainA.GetContext(), "07-tendermint-1005", "denom-b")
	s.Require().NoError(err, "unexpected error removing all pending receive packets - channel %s", "07-tendermint-1005")

	// Check that only the remaining sequences are found
	for _, channelID := range channels {
		for _, denom := range denoms {
			for sequence := range uint64(5) {
				removed := (denom == "denom-a" && sequence == 0) || (channelID == "channel-1" && denom == "denom-a") || (channelID == "07-tendermint-1005" && denom == "denom-b")
				actual, err := s.chainA.GetSimApp().RateLimitKeeper.CheckPacketReceivedDuringCurrentQuota(s.chainA.GetContext(), channelID, sequence, denom)
				s.Require().NoError(err, "unexpected error checking packet received during current quota - channel %s, sequence %s, denom %s", channelID, sequence, denom)

				// Assert that if we did not remove the packet, then we
				// successfully find it when checking the quota
				s.Require().Equal(!removed, actual, "receive packet after removal - channel: %s, sequence: %d, denom: %s", channelID, sequence, denom)
			}
		}
	}
}
