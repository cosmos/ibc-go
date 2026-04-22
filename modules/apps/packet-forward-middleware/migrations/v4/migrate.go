package v4

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"

	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/types"
)

const nonrefundableFieldNum = protowire.Number(12)

// Migrate migrates the x/packetforward module state from consensus version 3 to version 4.
// It removes the deprecated nonrefundable field from stored in-flight packets and aborts if
// any packet has nonrefundable=true.
func Migrate(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec) error {
	store := storeService.OpenKVStore(ctx)

	itr, err := store.Iterator(nil, nil)
	if err != nil {
		return err
	}
	defer itr.Close()

	for ; itr.Valid(); itr.Next() {
		fieldFound, fieldValue, err := readNonrefundableField(itr.Value())
		if err != nil {
			return fmt.Errorf("failed to decode in-flight packet %q: %w", string(itr.Key()), err)
		}

		if fieldFound && fieldValue {
			return fmt.Errorf("nonrefundable in-flight packet found during migration for key %q", string(itr.Key()))
		}

		var inFlightPacket types.InFlightPacket
		cdc.MustUnmarshal(itr.Value(), &inFlightPacket)

		updatedBz := cdc.MustMarshal(&inFlightPacket)
		if err := store.Set(itr.Key(), updatedBz); err != nil {
			return err
		}
	}

	return nil
}

func readNonrefundableField(bz []byte) (bool, bool, error) {
	for len(bz) > 0 {
		num, typ, n := protowire.ConsumeTag(bz)
		if n < 0 {
			return false, false, protowire.ParseError(n)
		}
		bz = bz[n:]

		if num == nonrefundableFieldNum && typ == protowire.VarintType {
			v, m := protowire.ConsumeVarint(bz)
			if m < 0 {
				return false, false, protowire.ParseError(m)
			}
			return true, v != 0, nil
		}

		m := protowire.ConsumeFieldValue(num, typ, bz)
		if m < 0 {
			return false, false, protowire.ParseError(m)
		}
		bz = bz[m:]
	}

	return false, false, nil
}
