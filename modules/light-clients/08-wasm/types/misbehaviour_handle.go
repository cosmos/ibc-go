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
	ClientMessage clientMessageConcretePayloadClientMessage `json:"client_message"`
}

func (c ClientState) CheckForMisbehaviour(ctx sdk.Context, _ codec.BinaryCodec, clientStore sdk.KVStore, msg exported.ClientMessage) bool {
	clientMsgConcrete := clientMessageConcretePayloadClientMessage{
		Header:       nil,
		Misbehaviour: nil,
	}
	switch clientMsg := msg.(type) {
	case *Header:
		clientMsgConcrete.Header = clientMsg
	case *Misbehaviour:
		clientMsgConcrete.Misbehaviour = clientMsg
	}

	if clientMsgConcrete.Header == nil && clientMsgConcrete.Misbehaviour == nil {
		return false
	}

	inner := checkForMisbehaviourInnerPayload{
		ClientMessage: clientMsgConcrete,
	}
	payload := checkForMisbehaviourPayload{
		CheckForMisbehaviour: inner,
	}

	result, err := call[contractResult](payload, &c, ctx, clientStore)
	if err != nil {
		panic(err)
	}

	return result.FoundMisbehaviour
}
