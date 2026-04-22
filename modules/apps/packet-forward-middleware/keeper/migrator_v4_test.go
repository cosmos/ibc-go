package keeper_test

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/runtime"

	pfmkeeper "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/keeper"
	"github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/migrations/v4/legacy"
	pfmtypes "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/types"
)

func (s *KeeperTestSuite) TestMigrate3to4() {
	tests := []struct {
		name        string
		nonref      bool
		expectError bool
	}{
		{
			name:   "migrates packets with nonrefundable false",
			nonref: false,
		},
		{
			name:        "errors on nonrefundable true",
			nonref:      true,
			expectError: true,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()

			ctx := s.chainA.GetContext()
			keeper := s.chainA.GetSimApp().PFMKeeper

			packet := &pfmtypes.InFlightPacket{
				PacketData:             []byte{1},
				OriginalSenderAddress:  s.chainA.SenderAccount.GetAddress().String(),
				RefundChannelId:        "channel-9",
				RefundPortId:           "transfer",
				PacketSrcChannelId:     "channel-0",
				PacketSrcPortId:        "transfer",
				PacketTimeoutTimestamp: 100,
				PacketTimeoutHeight:    "0-10",
				RefundSequence:         7,
				RetriesRemaining:       1,
				Timeout:                1000,
			}

			err := keeper.SetInflightPacket(ctx, packet.PacketSrcChannelId, packet.PacketSrcPortId, packet.RefundSequence, packet)
			s.Require().NoError(err)

			key := pfmtypes.RefundPacketKey(packet.PacketSrcChannelId, packet.PacketSrcPortId, packet.RefundSequence)
			storeService := runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(pfmtypes.StoreKey))
			store := storeService.OpenKVStore(ctx)

			legacyInFlightPacket := legacy.InFlightPacket{
				PacketData:             packet.PacketData,
				OriginalSenderAddress:  packet.OriginalSenderAddress,
				RefundChannelId:        packet.RefundChannelId,
				RefundPortId:           packet.RefundPortId,
				PacketSrcChannelId:     packet.PacketSrcChannelId,
				PacketSrcPortId:        packet.PacketSrcPortId,
				PacketTimeoutTimestamp: packet.PacketTimeoutTimestamp,
				PacketTimeoutHeight:    packet.PacketTimeoutHeight,
				RefundSequence:         packet.RefundSequence,
				RetriesRemaining:       packet.RetriesRemaining,
				Timeout:                packet.Timeout,
				Nonrefundable:          tc.nonref,
			}

			rawBz, err := legacyInFlightPacket.Marshal()
			s.Require().NoError(err)
			err = store.Set(key, rawBz)
			s.Require().NoError(err)

			migrator := pfmkeeper.NewMigrator(keeper)
			err = migrator.Migrate3to4(ctx)

			if tc.expectError {
				s.Require().ErrorContains(err, fmt.Sprintf("%q", string(key)))
				return
			}

			s.Require().NoError(err)

			updatedBz, err := store.Get(key)
			s.Require().NoError(err)

			var postMigrationLegacyPacket legacy.InFlightPacket
			err = postMigrationLegacyPacket.Unmarshal(updatedBz)
			s.Require().NoError(err)
			s.Require().False(postMigrationLegacyPacket.Nonrefundable)
		})
	}
}
