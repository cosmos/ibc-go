package keeper_test

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"

	"github.com/cosmos/cosmos-sdk/runtime"

	pfmkeeper "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/keeper"
	pfmtypes "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/types"
)

func (s *KeeperTestSuite) TestMigrate3to4() {
	tests := []struct {
		name        string
		fieldValue  byte
		expectError bool
	}{
		{
			name:       "migrates packets with nonrefundable false",
			fieldValue: 0x00,
		},
		{
			name:        "errors on nonrefundable true",
			fieldValue:  0x01,
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

			rawBz, err := store.Get(key)
			s.Require().NoError(err)
			rawBz = append(rawBz, 0x60, tc.fieldValue)
			err = store.Set(key, rawBz)
			s.Require().NoError(err)

			found, value, err := readLegacyNonrefundableField(rawBz)
			s.Require().NoError(err)
			s.Require().True(found)
			s.Require().Equal(tc.fieldValue == 0x01, value)

			migrator := pfmkeeper.NewMigrator(keeper)
			err = migrator.Migrate3to4(ctx)

			if tc.expectError {
				s.Require().ErrorContains(err, fmt.Sprintf("%q", string(key)))
				return
			}

			s.Require().NoError(err)

			updatedBz, err := store.Get(key)
			s.Require().NoError(err)

			found, _, err = readLegacyNonrefundableField(updatedBz)
			s.Require().NoError(err)
			s.Require().False(found)
		})
	}
}

func readLegacyNonrefundableField(bz []byte) (bool, bool, error) {
	for len(bz) > 0 {
		num, typ, n := protowire.ConsumeTag(bz)
		if n < 0 {
			return false, false, protowire.ParseError(n)
		}
		bz = bz[n:]

		if num == 12 && typ == protowire.VarintType {
			value, m := protowire.ConsumeVarint(bz)
			if m < 0 {
				return false, false, protowire.ParseError(m)
			}
			return true, value != 0, nil
		}

		m := protowire.ConsumeFieldValue(num, typ, bz)
		if m < 0 {
			return false, false, protowire.ParseError(m)
		}
		bz = bz[m:]
	}

	return false, false, nil
}
