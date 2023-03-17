package types

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
	ClientState   ClientState                               `json:"client_state"`
}

// VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
// It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
// will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
// if the ClientMessage fails to verify.
func (c ClientState) VerifyClientMessage(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) error {
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
		ClientState:   c,
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
	ClientState   ClientState                  `json:"client_state"`
}

type clientMessageConcretePayload struct {
	Header       *Header       `json:"header,omitempty"`
	Misbehaviour *Misbehaviour `json:"misbehaviour,omitempty"`
}

func (c ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	var clientMsgConcrete clientMessageConcretePayload
	switch clientMsg := clientMsg.(type) {
	case *Header:
		clientMsgConcrete.Header = clientMsg
	case *Misbehaviour:
		clientMsgConcrete.Misbehaviour = clientMsg
	}
	payload := updateStatePayload{
		UpdateState: updateStateInnerPayload{
			ClientMessage: clientMsgConcrete,
			ClientState:   c,
		},
	}

	output, err := call[contractResult](payload, &c, ctx, clientStore)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(output.Data, &c); err != nil {
		panic(sdkerrors.Wrapf(ErrUnableToUnmarshalPayload, fmt.Sprintf("underlying error: %s", err.Error())))
	}

	setClientState(clientStore, cdc, &c)

	return []exported.Height{c.LatestHeight}
}

type updateStateOnMisbehaviourPayload struct {
	UpdateStateOnMisbehaviour updateStateOnMisbehaviourInnerPayload `json:"update_state_on_misbehaviour"`
}
type updateStateOnMisbehaviourInnerPayload struct {
	ClientState   exported.ClientState   `json:"client_state"`
	ClientMessage exported.ClientMessage `json:"client_message"`
}

// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified
func (c ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) {
	payload := updateStateOnMisbehaviourPayload{
		UpdateStateOnMisbehaviour: updateStateOnMisbehaviourInnerPayload{
			ClientState:   &c,
			ClientMessage: clientMsg,
		},
	}
	output, err := call[contractResult](payload, &c, ctx, clientStore)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(output.Data, &c); err != nil {
		panic(sdkerrors.Wrapf(ErrUnableToUnmarshalPayload, fmt.Sprintf("underlying error: %s", err.Error())))
	}

	setClientState(clientStore, cdc, &c)
}