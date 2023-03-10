package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

type checkForMisbehaviourPayload struct {
	CheckForMisbehaviour checkForMisbehaviourInnerPayload `json:"check_for_misbehaviour"`
}
type checkForMisbehaviourInnerPayload struct {
	ClientState  exported.ClientState `json:"client_state"`
	Misbehaviour *Misbehaviour        `json:"misbehaviour"`
}

func (c ClientState) CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, msg exported.ClientMessage) bool {
	wasmMisbehaviour, ok := msg.(*Misbehaviour)
	if !ok {
		return false
	}

	payload := checkForMisbehaviourPayload{
		CheckForMisbehaviour: checkForMisbehaviourInnerPayload{
			ClientState:  &c,
			Misbehaviour: wasmMisbehaviour,
		},
	}

	_, err := call[contractResult](payload, &c, ctx, clientStore)
	if err != nil {
		panic(err)
	}

	return true
}