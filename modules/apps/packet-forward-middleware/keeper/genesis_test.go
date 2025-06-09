package keeper_test

import "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"

func (s *KeeperTestSuite) TestGenesis() {
	sampleInflight := types.InFlightPacket{
		PacketData:            []byte{1},
		OriginalSenderAddress: "senderAddress",
		RefundChannelId:       "refundChainID",
		RefundPortId:          "refundPortID",
		RefundSequence:        1,
		PacketSrcPortId:       "SourcePort",
		PacketSrcChannelId:    "SourceChannel",

		PacketTimeoutTimestamp: 1010101010,
		PacketTimeoutHeight:    "16-200",

		RetriesRemaining: 2,
		Timeout:          10101010101,
		Nonrefundable:    false,
	}

	key := types.RefundPacketKey(sampleInflight.PacketSrcChannelId, sampleInflight.PacketSrcPortId, sampleInflight.RefundSequence)
	keeper := s.chainA.GetSimApp().PFMKeeper
	err := keeper.SetInflightPacket(s.chainA.GetContext(), sampleInflight.PacketSrcChannelId, sampleInflight.PacketSrcPortId, sampleInflight.RefundSequence, &sampleInflight)
	s.Require().NoError(err)

	genState := keeper.ExportGenesis(s.chainA.GetContext())
	s.Require().Len(genState.InFlightPackets, 1)

	genesisInflight := genState.InFlightPackets[string(key)]

	s.Require().Equal(genesisInflight, sampleInflight)

	keeper.RemoveInFlightPacket(s.chainA.GetContext(), sampleInflight.ChannelPacket())
	inflightFromStore, err := keeper.GetInflightPacket(s.chainA.GetContext(), sampleInflight.ChannelPacket())
	s.Require().NoError(err)
	s.Require().Nil(inflightFromStore)

	keeper.InitGenesis(s.chainA.GetContext(), *genState)

	inflightFromStore, err = keeper.GetInflightPacket(s.chainA.GetContext(), sampleInflight.ChannelPacket())
	s.Require().NoError(err)
	s.Require().Equal(sampleInflight, *inflightFromStore)
}
