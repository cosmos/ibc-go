package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

type (
	checkForMisbehaviourInnerPayload struct {
		ClientMessage *ClientMessage `json:"client_message"`
	}
	checkForMisbehaviourPayload struct {
		CheckForMisbehaviour checkForMisbehaviourInnerPayload `json:"check_for_misbehaviour"`
	}
)

// CheckForMisbehaviour detects misbehaviour in a submitted Header message and verifies
// the correctness of a submitted Misbehaviour ClientMessage
func (cs ClientState) CheckForMisbehaviour(ctx sdk.Context, _ codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) bool {
	clientMessage, ok := clientMsg.(*ClientMessage)
	if !ok {
		return false
	}

	payload := checkForMisbehaviourPayload{
		CheckForMisbehaviour: checkForMisbehaviourInnerPayload{
			ClientMessage: clientMessage,
		},
	}

	result, err := wasmQuery[checkForMisbehaviourResult](ctx, clientStore, &cs, payload)
	if err != nil {
		panic(err)
	}

	return result.FoundMisbehaviour
}
