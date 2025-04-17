package simulation

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/kv"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// NewDecodeStore returns a decoder function closure that unmarshals the KVPair's
// Value to the corresponding rate-limiting types.
func NewDecodeStore() func(kvA, kvB kv.Pair) string {
	return func(kvA, kvB kv.Pair) string {
		switch {
		case bytes.Equal(kvA.Key[:len([]byte(types.PortKeyPrefix))], []byte(types.PortKeyPrefix)):
			return fmt.Sprintf("Port A: %s\nPort B: %s", string(kvA.Value), string(kvB.Value))

		case bytes.Equal(kvA.Key[:len([]byte(types.RateLimitKeyPrefix))], []byte(types.RateLimitKeyPrefix)):
			var flowA, flowB types.Flow
			types.ModuleCdc.MustUnmarshal(kvA.Value, &flowA)
			types.ModuleCdc.MustUnmarshal(kvB.Value, &flowB)
			return fmt.Sprintf("Flow A: %s\nFlow B: %s", flowA.String(), flowB.String())

		case bytes.Equal(kvA.Key[:len([]byte(types.PendingSendPacketPrefix))], []byte(types.PendingSendPacketPrefix)):
			return fmt.Sprintf("PendingSendPacket A: %s\nPendingSendPacket B: %s", string(kvA.Value), string(kvB.Value))

		case bytes.Equal(kvA.Key[:len([]byte(types.DenomBlacklistKeyPrefix))], []byte(types.DenomBlacklistKeyPrefix)):
			return fmt.Sprintf("DenomBlacklist A: %s\nDenomBlacklist B: %s", string(kvA.Value), string(kvB.Value))

		case bytes.Equal(kvA.Key[:len([]byte(types.AddressWhitelistKeyPrefix))], []byte(types.AddressWhitelistKeyPrefix)):
			return fmt.Sprintf("AddressWhitelist A: %s\nAddressWhitelist B: %s", string(kvA.Value), string(kvB.Value))

		case bytes.Equal(kvA.Key[:len([]byte(types.HourEpochKey))], []byte(types.HourEpochKey)):
			return fmt.Sprintf("HourEpoch A: %s\nHourEpoch B: %s", string(kvA.Value), string(kvB.Value))

		default:
			panic(fmt.Errorf("invalid %s key prefix %X", types.ModuleName, kvA.Key[:1]))
		}
	}
}
