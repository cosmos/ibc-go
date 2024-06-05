package simulation

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/kv"

	internaltypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/internal/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

// NewDecodeStore returns a decoder function closure that unmarshals the KVPair's
// Value to the corresponding DenomTrace type.
func NewDecodeStore() func(kvA, kvB kv.Pair) string {
	return func(kvA, kvB kv.Pair) string {
		switch {
		case bytes.Equal(kvA.Key[:1], types.PortKey):
			return fmt.Sprintf("Port A: %s\nPort B: %s", string(kvA.Value), string(kvB.Value))

		case bytes.Equal(kvA.Key[:1], types.DenomTraceKey):
			var denomTraceA, denomTraceB internaltypes.DenomTrace
			types.ModuleCdc.MustUnmarshal(kvA.Value, &denomTraceA)
			types.ModuleCdc.MustUnmarshal(kvB.Value, &denomTraceB)
			return fmt.Sprintf("DenomTrace A: %s\nDenomTrace B: %s", denomTraceA.IBCDenom(), denomTraceB.IBCDenom())

		default:
			panic(fmt.Errorf("invalid %s key prefix %X", types.ModuleName, kvA.Key[:1]))
		}
	}
}
