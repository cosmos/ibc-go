package keeper_test

import "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"

func (s *KeeperTestSuite) TestGenesis() {
	inFlightPacket := &types.InFlightPacket{
		PacketData:            []byte{1},
		OriginalSenderAddress: "senderAddress",
		RefundChannelId:       "refundChainID",
		RefundPortId:          "refundPortID",
		RefundSequence:        1,
		PacketSrcPortId:       "SourcePort",
		PacketSrcChannelId:    "SourceChannel",

		PacketTimeoutTimestamp: 1010101010,
		PacketTimeoutHeight:    "100",

		RetriesRemaining: 2,
		Timeout:          10101010101,
		Nonrefundable:    false,
	}

	key := types.RefundPacketKey("chanID", "portID", 1)
	keeper := s.chainA.GetSimApp().PFMKeeper
	err := keeper.SetInflightPacket(s.chainA.GetContext(), "chanID", "portID", 1, inFlightPacket)
	s.Require().NoError(err)

	genState := keeper.ExportGenesis(s.chainA.GetContext())
	s.Require().Len(genState.InFlightPackets, 1)

	genesisInflight := genState.InFlightPackets[string(key)]

	s.Require().Equal(genesisInflight, *inFlightPacket)

}
