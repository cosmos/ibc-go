package simulation

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/kv"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// NewDecodeStore returns a decoder function closure that unmarshals the KVPair's
// Value to the corresponding client type.
func NewDecodeStore(cdc codec.BinaryCodec, kvA, kvB kv.Pair) (string, bool) {
	switch {
	case bytes.HasPrefix(kvA.Key, host.KeyClientStorePrefix) && bytes.HasSuffix(kvA.Key, host.ClientStateKey()):
		clientStateA := types.MustUnmarshalClientState(cdc, kvA.Value)
		clientStateB := types.MustUnmarshalClientState(cdc, kvB.Value)
		return fmt.Sprintf("ClientState A: %v\nClientState B: %v", clientStateA, clientStateB), true

	case bytes.HasPrefix(kvA.Key, host.KeyClientStorePrefix) && bytes.Contains(kvA.Key, []byte(host.KeyConsensusStatePrefix)):
		consensusStateA := types.MustUnmarshalConsensusState(cdc, kvA.Value)
		consensusStateB := types.MustUnmarshalConsensusState(cdc, kvB.Value)
		return fmt.Sprintf("ConsensusState A: %v\nConsensusState B: %v", consensusStateA, consensusStateB), true

	default:
		return "", false
	}
}
