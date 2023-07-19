package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

// VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
// It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
// will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
// if the ClientMessage fails to verify.
func (cs ClientState) VerifyClientMessage(ctx sdk.Context, _ codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) error {
	verifyClientMessageMsg := verifyClientMessageMsg{
		Header:       nil,
		Misbehaviour: nil,
	}
	switch clientMsg := clientMsg.(type) {
	case *Header:
		verifyClientMessageMsg.Header = clientMsg
	case *Misbehaviour:
		verifyClientMessageMsg.Misbehaviour = clientMsg
	}
	payload := QueryMsg{
		VerifyClientMessage: &verifyClientMessageMsg,
	}
	_, err := call[contractResult](ctx, clientStore, &cs, payload)
	return err
}

// Client state and new consensus states are updated in the store by the contract
func (cs ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	header, ok := clientMsg.(*Header)
	if !ok {
		panic(fmt.Errorf("expected type %T, got %T", &Header{}, clientMsg))
	}

	payload := SudoMsg{
		UpdateState: &updateStateMsg{Header: header},
	}

	_, err := call[contractResult](ctx, clientStore, &cs, payload)
	if err != nil {
		panic(err)
	}

	return []exported.Height{clientMsg.(*Header).Height}
}

// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified
// Client state is updated in the store by contract.
func (cs ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, _ codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) {
	var updateStateOnMisbehaviourMsg updateStateOnMisbehaviourMsg
	switch clientMsg := clientMsg.(type) {
	case *Header:
		updateStateOnMisbehaviourMsg.Header = clientMsg
	case *Misbehaviour:
		updateStateOnMisbehaviourMsg.Misbehaviour = clientMsg
	}

	payload := SudoMsg{
		UpdateStateOnMisbehaviour: &updateStateOnMisbehaviourMsg,
	}

	_, err := call[contractResult](ctx, clientStore, &cs, payload)
	if err != nil {
		panic(err)
	}
}
