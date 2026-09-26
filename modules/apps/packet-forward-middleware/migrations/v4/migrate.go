// SPDX-License-Identifier: Apache-2.0

package v4

import (
	"fmt"

	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/migrations/v4/legacy"
	"github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
)

// LegacyParamsKey is the store key the upstream packetforward middleware used for
// its params (ParamsKey in packetforward/types/keys.go, removed by cosmos/ibc-apps#202).
var LegacyParamsKey = []byte{0x00}

// Migrate migrates the x/packetforward module state from consensus version 3 to version 4.
// It removes the deprecated nonrefundable field from stored in-flight packets and aborts if
// any packet has nonrefundable=true.
func Migrate(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec) error {
	store := storeService.OpenKVStore(ctx)

	if err := store.Delete(LegacyParamsKey); err != nil {
		return fmt.Errorf("failed to delete legacy PFM params key: %w", err)
	}

	itr, err := store.Iterator(nil, nil)
	if err != nil {
		return err
	}
	defer itr.Close()

	for ; itr.Valid(); itr.Next() {
		var legacyInFlightPacket legacy.InFlightPacket
		err = legacyInFlightPacket.Unmarshal(itr.Value())
		if err != nil {
			return fmt.Errorf("failed to unmarshal legacy in-flight packet for key %q: %w", string(itr.Key()), err)
		}

		if legacyInFlightPacket.Nonrefundable {
			return fmt.Errorf("nonrefundable in-flight packet found during migration for key %q", string(itr.Key()))
		}

		inFlightPacket := types.InFlightPacket{
			OriginalSenderAddress:  legacyInFlightPacket.OriginalSenderAddress,
			RefundChannelId:        legacyInFlightPacket.RefundChannelId,
			RefundPortId:           legacyInFlightPacket.RefundPortId,
			PacketSrcChannelId:     legacyInFlightPacket.PacketSrcChannelId,
			PacketSrcPortId:        legacyInFlightPacket.PacketSrcPortId,
			PacketTimeoutTimestamp: legacyInFlightPacket.PacketTimeoutTimestamp,
			PacketTimeoutHeight:    legacyInFlightPacket.PacketTimeoutHeight,
			PacketData:             legacyInFlightPacket.PacketData,
			RefundSequence:         legacyInFlightPacket.RefundSequence,
			RetriesRemaining:       legacyInFlightPacket.RetriesRemaining,
			Timeout:                legacyInFlightPacket.Timeout,
		}

		if _, err := clienttypes.ParseHeight(inFlightPacket.PacketTimeoutHeight); err != nil {
			return fmt.Errorf("invalid in-flight packet found during migration for key %q: %w", string(itr.Key()), err)
		}

		if err := inFlightPacket.ChannelPacket().ValidateBasic(); err != nil {
			return fmt.Errorf("invalid in-flight packet found during migration for key %q: %w", string(itr.Key()), err)
		}

		updatedBz := cdc.MustMarshal(&inFlightPacket)
		if err := store.Set(itr.Key(), updatedBz); err != nil {
			return err
		}
	}

	return nil
}
