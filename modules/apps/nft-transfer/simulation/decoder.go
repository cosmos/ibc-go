package simulation

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/kv"

	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
)

// TransferUnmarshaler defines the expected encoding store functions.
type TransferUnmarshaler interface {
	MustUnmarshalClassTrace([]byte) types.ClassTrace
}

// NewDecodeStore returns a decoder function closure that unmarshals the KVPair's
// Value to the corresponding ClassTrace type.
func NewDecodeStore(cdc TransferUnmarshaler) func(kvA, kvB kv.Pair) string {
	return func(kvA, kvB kv.Pair) string {
		switch {
		case bytes.Equal(kvA.Key[:1], types.PortKey):
			return fmt.Sprintf("Port A: %s\nPort B: %s", string(kvA.Value), string(kvB.Value))

		case bytes.Equal(kvA.Key[:1], types.ClassTraceKey):
			classTraceA := cdc.MustUnmarshalClassTrace(kvA.Value)
			classTraceB := cdc.MustUnmarshalClassTrace(kvB.Value)
			return fmt.Sprintf("ClassTrace A: %s\nClassTrace B: %s", classTraceA.IBCClassID(), classTraceB.IBCClassID())

		default:
			panic(fmt.Sprintf("invalid %s key prefix %X", types.ModuleName, kvA.Key[:1]))
		}
	}
}
