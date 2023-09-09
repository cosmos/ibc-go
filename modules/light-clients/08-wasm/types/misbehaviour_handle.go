package types

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// CheckForMisbehaviour detects misbehaviour in a submitted Header message and verifies
// the correctness of a submitted Misbehaviour ClientMessage
func (cs ClientState) CheckForMisbehaviour(ctx sdk.Context, _ codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) bool {
	clientMessage, ok := clientMsg.(*ClientMessage)
	if !ok {
		return false
	}

	payload := queryMsg{
		CheckForMisbehaviour: &checkForMisbehaviourMsg{ClientMessage: clientMessage},
	}

	result, err := wasmQuery[checkForMisbehaviourResult](ctx, clientStore, &cs, payload)
	if err != nil {
		panic(err)
	}

	return result.FoundMisbehaviour
}
