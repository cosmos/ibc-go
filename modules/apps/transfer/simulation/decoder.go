package simulation

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/kv"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

// NewDecodeStore returns a decoder function closure that unmarshals the KVPair's
// Value to the corresponding Denom type.
func NewDecodeStore() func(kvA, kvB kv.Pair) string {
	return func(kvA, kvB kv.Pair) string {
		switch {
		case bytes.Equal(kvA.Key[:1], types.PortKey):
			return fmt.Sprintf("Port A: %s\nPort B: %s", string(kvA.Value), string(kvB.Value))

		case bytes.Equal(kvA.Key[:1], types.DenomKey):
			var denomA, denomB types.Denom
			types.ModuleCdc.MustUnmarshal(kvA.Value, &denomA)
			types.ModuleCdc.MustUnmarshal(kvB.Value, &denomB)
			return fmt.Sprintf("Denom A: %s\nDenom B: %s", denomA.IBCDenom(), denomB.IBCDenom())

		default:
			panic(fmt.Errorf("invalid %s key prefix %X", types.ModuleName, kvA.Key[:1]))
		}
	}
}
