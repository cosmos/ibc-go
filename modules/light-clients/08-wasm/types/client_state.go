package types

import (
	"encoding/hex"
	"encoding/json"
	"errors"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

// NewClientState creates a new ClientState instance.
func NewClientState(data []byte, codeHash []byte, height clienttypes.Height) *ClientState {
	return &ClientState{
		Data:         data,
		CodeHash:     codeHash,
		LatestHeight: height,
	}
}

// ClientType is Wasm light client.
func (cs ClientState) ClientType() string {
	return exported.Wasm
}

// GetLatestHeight returns latest block height.
func (cs ClientState) GetLatestHeight() exported.Height {
	return cs.LatestHeight
}

// Validate performs a basic validation of the client state fields.
func (cs ClientState) Validate() error {
	if len(cs.Data) == 0 {
		return errorsmod.Wrap(ErrInvalidData, "data cannot be empty")
	}

	lenCodeHash := len(cs.CodeHash)
	if lenCodeHash == 0 {
		return errorsmod.Wrap(ErrInvalidCodeHash, "code hash cannot be empty")
	}
	if lenCodeHash != 32 { // sha256 output is 256 bits long
		return errorsmod.Wrapf(ErrInvalidCodeHash, "expected length of 32 bytes, got %d", lenCodeHash)
	}

	return nil
}

type (
	statusInnerPayload struct{}
	statusPayload      struct {
		Status statusInnerPayload `json:"status"`
	}
)

// Status returns the status of the wasm client.
// The client may be:
// - Active: frozen height is zero and client is not expired
// - Frozen: frozen height is not zero
// - Expired: the latest consensus state timestamp + trusting period <= current time
// - Unauthorized: the client type is not registered as an allowed client type
//
// A frozen client will become expired, so the Frozen status
// has higher precedence.
func (cs ClientState) Status(ctx sdk.Context, clientStore sdk.KVStore, _ codec.BinaryCodec) exported.Status {
	payload := statusPayload{Status: statusInnerPayload{}}

	encodedData, err := json.Marshal(payload)
	if err != nil {
		return exported.Unknown
	}

	response, err := queryContract(ctx, clientStore, cs.CodeHash, encodedData)
	if err != nil {
		return exported.Unknown
	}
	var output queryResponse
	if err := json.Unmarshal(response, &output); err != nil {
		return exported.Unknown
	}

	return output.Status
}

// ZeroCustomFields returns a ClientState that is a copy of the current ClientState
// with all client customizable fields zeroed out
func (cs ClientState) ZeroCustomFields() exported.ClientState {
	return &cs
}

func (cs ClientState) GetTimestampAtHeight(
	_ sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
) (uint64, error) {
	// get consensus state at height from clientStore to check for expiry
	consState, err := GetConsensusState(clientStore, cdc, height)
	if err != nil {
		return 0, errorsmod.Wrapf(err, "height (%s)", height)
	}
	return consState.GetTimestamp(), nil
}

type instantiateMessage struct {
	ClientState    *ClientState    `json:"client_state"`
	ConsensusState *ConsensusState `json:"consensus_state"`
}

// Initialize checks that the initial consensus state is an 08-wasm consensus state and
// sets the client state, consensus state in the provided client store.
// It also initializes the wasm contract for the client.
func (cs ClientState) Initialize(ctx sdk.Context, _ codec.BinaryCodec, clientStore sdk.KVStore, state exported.ConsensusState) error {
	consensusState, ok := state.(*ConsensusState)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrInvalidConsensus, "invalid initial consensus state. expected type: %T, got: %T",
			&ConsensusState{}, state)
	}

	payload := instantiateMessage{
		ClientState:    &cs,
		ConsensusState: consensusState,
	}

	encodedData, err := json.Marshal(payload)
	if err != nil {
		return sdkerrors.Wrapf(err, "failed to marshal payload for wasm contract instantiation")
	}

	_, err = initContract(ctx, clientStore, cs.CodeHash, encodedData)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to initialize contract")
	}
	return nil
}

type (
	verifyMembershipInnerPayload struct {
		Height           exported.Height `json:"height"`
		DelayTimePeriod  uint64          `json:"delay_time_period"`
		DelayBlockPeriod uint64          `json:"delay_block_period"`
		Proof            []byte          `json:"proof"`
		Path             exported.Path   `json:"path"`
		Value            []byte          `json:"value"`
	}
	verifyMembershipPayload struct {
		VerifyMembership verifyMembershipInnerPayload `json:"verify_membership"`
	}
)

// VerifyMembership is a generic proof verification method which verifies a proof of the existence of a value at a given CommitmentPath at the specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
// If a zero proof height is passed in, it will fail to retrieve the associated consensus state.
func (cs ClientState) VerifyMembership(
	ctx sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
	value []byte,
) error {
	if cs.GetLatestHeight().LT(height) {
		return errorsmod.Wrapf(
			ibcerrors.ErrInvalidHeight,
			"client state height < proof height (%d < %d), please ensure the client has been updated", cs.GetLatestHeight(), height,
		)
	}

	_, ok := path.(commitmenttypes.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypes.MerklePath{}, path)
	}

	_, err := GetConsensusState(clientStore, cdc, height)
	if err != nil {
		return errorsmod.Wrap(err, "please ensure the proof was constructed against a height that exists on the client")
	}

	payload := verifyMembershipPayload{
		VerifyMembership: verifyMembershipInnerPayload{
			Height:           height,
			DelayTimePeriod:  delayTimePeriod,
			DelayBlockPeriod: delayBlockPeriod,
			Proof:            proof,
			Path:             path,
			Value:            value,
		},
	}
	_, err = call[contractResult](ctx, clientStore, &cs, payload)
	return err
}

type (
	verifyNonMembershipInnerPayload struct {
		Height           exported.Height `json:"height"`
		DelayTimePeriod  uint64          `json:"delay_time_period"`
		DelayBlockPeriod uint64          `json:"delay_block_period"`
		Proof            []byte          `json:"proof"`
		Path             exported.Path   `json:"path"`
	}
	verifyNonMembershipPayload struct {
		VerifyNonMembership verifyNonMembershipInnerPayload `json:"verify_non_membership"`
	}
)

// VerifyNonMembership is a generic proof verification method which verifies the absence of a given CommitmentPath at a specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
// If a zero proof height is passed in, it will fail to retrieve the associated consensus state.
func (cs ClientState) VerifyNonMembership(
	ctx sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
) error {
	if cs.GetLatestHeight().LT(height) {
		return errorsmod.Wrapf(
			ibcerrors.ErrInvalidHeight,
			"client state height < proof height (%d < %d), please ensure the client has been updated", cs.GetLatestHeight(), height,
		)
	}

	_, ok := path.(commitmenttypes.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypes.MerklePath{}, path)
	}

	_, err := GetConsensusState(clientStore, cdc, height)
	if err != nil {
		return errorsmod.Wrap(err, "please ensure the proof was constructed against a height that exists on the client")
	}

	payload := verifyNonMembershipPayload{
		VerifyNonMembership: verifyNonMembershipInnerPayload{
			Height:           height,
			DelayTimePeriod:  delayTimePeriod,
			DelayBlockPeriod: delayBlockPeriod,
			Proof:            proof,
			Path:             path,
		},
	}
	_, err = call[contractResult](ctx, clientStore, &cs, payload)
	return err
}

// call calls the contract with the given payload and writes the result to output.
func call[T ContractResult](ctx sdk.Context, clientStore sdk.KVStore, cs *ClientState, payload any) (T, error) {
	var output T
	encodedData, err := json.Marshal(payload)
	if err != nil {
		return output, errorsmod.Wrapf(err, "failed to marshal wasm contract payload")
	}
	out, err := callContract(ctx, clientStore, cs.CodeHash, encodedData)
	if err != nil {
		return output, errorsmod.Wrapf(err, "call to wasm contract failed")
	}
	if err := json.Unmarshal(out.Data, &output); err != nil {
		return output, errorsmod.Wrapf(err, "failed unmarshal wasm contract payload")
	}
	if !output.Validate() {
		return output, errorsmod.Wrapf(errors.New(output.Error()), "error occurred while calling contract with code hash %s", hex.EncodeToString(cs.CodeHash))
	}
	if len(out.Messages) > 0 {
		return output, errorsmod.Wrapf(ErrWasmSubMessagesNotAllowed, "code hash (%s)", hex.EncodeToString(cs.CodeHash))
	}
	return output, nil
}
