package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

type (
	verifyClientMessageInnerPayload struct {
		ClientMessage *ClientMessage `json:"client_message"`
	}
	verifyClientMessagePayload struct {
		VerifyClientMessage verifyClientMessageInnerPayload `json:"verify_client_message"`
	}
)

// VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
// It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
// will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
// if the ClientMessage fails to verify.
func (cs ClientState) VerifyClientMessage(ctx sdk.Context, _ codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) error {
	clientMessage, ok := clientMsg.(*ClientMessage)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected type: %T, got: %T", &ClientMessage{}, clientMsg)
	}

	payload := verifyClientMessagePayload{
		VerifyClientMessage: verifyClientMessageInnerPayload{
			ClientMessage: clientMessage,
		},
	}
	_, err := wasmQuery[contractResult](ctx, clientStore, &cs, payload)
	return err
}

type (
	updateStateInnerPayload struct {
		ClientMessage *ClientMessage `json:"client_message"`
	}
	updateStatePayload struct {
		UpdateState updateStateInnerPayload `json:"update_state"`
	}
)

// Client state and new consensus states are updated in the store by the contract
func (cs ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	clientMessage, ok := clientMsg.(*ClientMessage)
	if !ok {
		panic(fmt.Errorf("expected type %T, got %T", &ClientMessage{}, clientMsg))
	}

	payload := updateStatePayload{
		UpdateState: updateStateInnerPayload{
			ClientMessage: clientMessage,
		},
	}

	result, err := call[updateStateResult](ctx, clientStore, &cs, payload)
	if err != nil {
		panic(err)
	}

	return result.Heights
}

type (
	updateStateOnMisbehaviourInnerPayload struct {
		ClientMessage *ClientMessage `json:"client_message"`
	}
	updateStateOnMisbehaviourPayload struct {
		UpdateStateOnMisbehaviour updateStateOnMisbehaviourInnerPayload `json:"update_state_on_misbehaviour"`
	}
)

// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified
// Client state is updated in the store by contract.
func (cs ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, _ codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) {
	clientMessage, ok := clientMsg.(*ClientMessage)
	if !ok {
		panic(fmt.Errorf("expected type %T, got %T", &ClientMessage{}, clientMsg))
	}

	payload := updateStateOnMisbehaviourPayload{
		UpdateStateOnMisbehaviour: updateStateOnMisbehaviourInnerPayload{
			ClientMessage: clientMessage,
		},
	}

	_, err := call[contractResult](ctx, clientStore, &cs, payload)
	if err != nil {
		panic(err)
	}
}
