package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// CheckForMisbehaviour detects misbehaviour in a submitted Header message and verifies
// the correctness of a submitted Misbehaviour ClientMessage
func (cs ClientState) CheckForMisbehaviour(ctx sdk.Context, _ codec.BinaryCodec, clientStore sdk.KVStore, msg exported.ClientMessage) bool {
	checkForMisbehaviourMsg := checkForMisbehaviourMsg{
		Header:       nil,
		Misbehaviour: nil,
	}
	switch clientMsg := msg.(type) {
	case *Header:
		checkForMisbehaviourMsg.Header = clientMsg
	case *Misbehaviour:
		checkForMisbehaviourMsg.Misbehaviour = clientMsg
	}

	if checkForMisbehaviourMsg.Header == nil && checkForMisbehaviourMsg.Misbehaviour == nil {
		return false
	}

	payload := QueryMsg{
		CheckForMisbehaviour: &checkForMisbehaviourMsg,
	}

	result, err := call[contractResult](ctx, clientStore, &cs, payload)
	if err != nil {
		panic(err)
	}

	return result.FoundMisbehaviour
}
