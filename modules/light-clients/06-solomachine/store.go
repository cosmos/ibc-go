package solomachine

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// getClientState retrieves the client state from the store using the provided KVStore and codec.
// It returns the unmarshaled ClientState and a boolean indicating if the state was found.
func getClientState(store storetypes.KVStore, cdc codec.BinaryCodec) (*ClientState, bool) {
	bz := store.Get(host.ClientStateKey())
	if len(bz) == 0 {
		return nil, false
	}

	clientStateI := clienttypes.MustUnmarshalClientState(cdc, bz)
	var clientState *ClientState
	clientState, ok := clientStateI.(*ClientState)
	if !ok {
		panic(fmt.Errorf("cannot convert %T to %T", clientStateI, clientState))
	}

	return clientState, true
}
