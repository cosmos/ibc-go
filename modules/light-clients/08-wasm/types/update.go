package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

type verifyClientMessagePayload struct {
	VerifyClientMessage verifyClientMessageInnerPayload `json:"verify_client_message"`
}

type clientMessageConcretePayloadClientMessage struct {
	Header       *Header       `json:"header,omitempty"`
	Misbehaviour *Misbehaviour `json:"misbehaviour,omitempty"`
}
type verifyClientMessageInnerPayload struct {
	ClientMessage clientMessageConcretePayloadClientMessage `json:"client_message"`
}

// VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
// It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
// will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
// if the ClientMessage fails to verify.
func (c ClientState) VerifyClientMessage(ctx sdk.Context, _ codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) error {
	clientMsgConcrete := clientMessageConcretePayloadClientMessage{
		Header:       nil,
		Misbehaviour: nil,
	}
	switch clientMsg := clientMsg.(type) {
	case *Header:
		clientMsgConcrete.Header = clientMsg
	case *Misbehaviour:
		clientMsgConcrete.Misbehaviour = clientMsg
	}
	inner := verifyClientMessageInnerPayload{
		ClientMessage: clientMsgConcrete,
	}
	payload := verifyClientMessagePayload{
		VerifyClientMessage: inner,
	}
	_, err := call[contractResult](payload, &c, ctx, clientStore)
	return err
}

type updateStatePayload struct {
	UpdateState updateStateInnerPayload `json:"update_state"`
}
type updateStateInnerPayload struct {
	ClientMessage clientMessageConcretePayload `json:"client_message"`
}

type clientMessageConcretePayload struct {
	Header       *Header       `json:"header,omitempty"`
	Misbehaviour *Misbehaviour `json:"misbehaviour,omitempty"`
}

// Client state and new consensus states are updated in the store by the contract
func (c ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, store sdk.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	header, ok := clientMsg.(*Header)
	if !ok {
		panic(fmt.Errorf("expected type %T, got %T", &Header{}, clientMsg))
	}
	
	payload := updateStatePayload{
		UpdateState: updateStateInnerPayload{
			ClientMessage: clientMessageConcretePayload{
				Header: header,
			},
		},
	}

	_, err := call[contractResult](payload, &c, ctx, store)
	if err != nil {
		panic(err)
	}

	return []exported.Height{clientMsg.(*Header).Height}
}

type updateStateOnMisbehaviourPayload struct {
	UpdateStateOnMisbehaviour updateStateOnMisbehaviourInnerPayload `json:"update_state_on_misbehaviour"`
}
type updateStateOnMisbehaviourInnerPayload struct {
	ClientMessage clientMessageConcretePayloadClientMessage `json:"client_message"`
}

// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified
// Client state is updated in the store by contract
func (c ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, _ codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) {
	clientMsgConcrete := clientMessageConcretePayloadClientMessage{
		Header:       nil,
		Misbehaviour: nil,
	}
	switch clientMsg := clientMsg.(type) {
	case *Header:
		clientMsgConcrete.Header = clientMsg
	case *Misbehaviour:
		clientMsgConcrete.Misbehaviour = clientMsg
	}

	inner := updateStateOnMisbehaviourInnerPayload{
		ClientMessage: clientMsgConcrete,
	}

	payload := updateStateOnMisbehaviourPayload{
		UpdateStateOnMisbehaviour: inner,
	}

	_, err := call[contractResult](payload, &c, ctx, clientStore)
	if err != nil {
		panic(err)
	}
}
