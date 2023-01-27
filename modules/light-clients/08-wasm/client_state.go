package wasm

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

func (c ClientState) ClientType() string {
	return exported.Wasm
}

func (c ClientState) GetLatestHeight() exported.Height {
	return c.LatestHeight
}

func (c ClientState) Validate() error {
	if c.Data == nil || len(c.Data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	if c.CodeId == nil || len(c.CodeId) == 0 {
		return fmt.Errorf("codeid cannot be empty")
	}

	return nil
}

// TODO call into the contract here to get the status once it is implemented in the contract, for now, returns active
func (c ClientState) Status(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec) exported.Status {
	return exported.Active
}

func (c ClientState) ExportMetadata(store sdk.KVStore) []exported.GenesisMetadata {
	const ExportMetadataQuery = "exportmetadata"
	payload := make(map[string]map[string]interface{})
	payload[ExportMetadataQuery] = make(map[string]interface{})
	inner := payload[ExportMetadataQuery]
	inner["client_state"] = c

	encodedData, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	response, err := queryContractWithStore(c.CodeId, store, encodedData)
	if err != nil {
		panic(err)
	}

	output := queryResponse{}
	if err := json.Unmarshal(response, &output); err != nil {
		panic(err)
	}

	genesisMetadata := make([]exported.GenesisMetadata, len(output.GenesisMetadata))
	for i, metadata := range output.GenesisMetadata {
		genesisMetadata[i] = metadata
	}
	return genesisMetadata
}

func (c ClientState) ZeroCustomFields() exported.ClientState {
	return &c
}

func (c ClientState) GetTimestampAtHeight(
	ctx sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
) (uint64, error) {
	// get consensus state at height from clientStore to check for expiry
	consState, found := GetConsensusState(clientStore, cdc, height)
	if found != nil {
		return 0, sdkerrors.Wrapf(clienttypes.ErrConsensusStateNotFound, "height (%s)", height)
	}
	return consState.GetTimestamp(), nil
}

func (c ClientState) Initialize(context sdk.Context, marshaler codec.BinaryCodec, store sdk.KVStore, state exported.ConsensusState) error {
	consensusState, ok := state.(*ConsensusState)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "invalid initial consensus state. expected type: %T, got: %T",
			&ConsensusState{}, state)
	}
	setClientState(store, marshaler, &c)
	setConsensusState(store, marshaler, consensusState, c.GetLatestHeight())

	_, err := initContract(c.CodeId, context, store)
	if err != nil {
		return sdkerrors.Wrapf(ErrUnableToInit, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	return nil
}

func (c ClientState) VerifyMembership(
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
	const VerifyClientMessage = "verify_membership"
	inner := make(map[string]interface{})
	inner["height"] = height
	inner["delay_time_period"] = delayTimePeriod
	inner["delay_block_period"] = delayBlockPeriod
	inner["proof"] = proof
	inner["path"] = path
	inner["value"] = value
	payload := make(map[string]map[string]interface{})
	payload[VerifyClientMessage] = inner

	_, err := call[contractResult](payload, &c, ctx, clientStore)
	return err
}

func (c ClientState) VerifyNonMembership(
	ctx sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
) error {
	const VerifyClientMessage = "verify_non_membership"
	inner := make(map[string]interface{})
	inner["height"] = height
	inner["delay_time_period"] = delayTimePeriod
	inner["delay_block_period"] = delayBlockPeriod
	inner["proof"] = proof
	inner["path"] = path
	payload := make(map[string]map[string]interface{})
	payload[VerifyClientMessage] = inner

	_, err := call[contractResult](payload, &c, ctx, clientStore)
	return err
}

// VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
// It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
// will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
// if the ClientMessage fails to verify.
func (c ClientState) VerifyClientMessage(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) error {
	const VerifyClientMessage = "verify_client_message"
	inner := make(map[string]interface{})
	clientMsgConcrete := make(map[string]interface{})
	switch clientMsg := clientMsg.(type) {
	case *Header:
		clientMsgConcrete["header"] = clientMsg
	case *Misbehaviour:
		clientMsgConcrete["misbehaviour"] = clientMsg
	}
	inner["client_message"] = clientMsgConcrete
	inner["client_state"] = c
	payload := make(map[string]map[string]interface{})
	payload[VerifyClientMessage] = inner

	_, err := call[contractResult](payload, &c, ctx, clientStore)
	return err
}

func (c ClientState) CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, msg exported.ClientMessage) bool {
	wasmMisbehaviour, ok := msg.(*Misbehaviour)
	if !ok {
		return false
	}

	const checkForMisbehaviourMessage = "check_for_misbehaviour"
	payload := make(map[string]map[string]interface{})
	payload[checkForMisbehaviourMessage] = make(map[string]interface{})
	inner := payload[checkForMisbehaviourMessage]
	inner["client_state"] = c
	inner["misbehaviour"] = wasmMisbehaviour

	_, err := call[contractResult](payload, &c, ctx, clientStore)
	if err != nil {
		panic(err)
	}

	return true
}

// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified
func (c ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) {
	const updateStateOnMisbehaviour = "update_state_on_misbehaviour"
	payload := make(map[string]map[string]interface{})
	payload[updateStateOnMisbehaviour] = make(map[string]interface{})
	inner := payload[updateStateOnMisbehaviour]
	inner["client_state"] = c
	inner["client_message"] = clientMsg

	encodedData, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, err = callContract(c.CodeId, ctx, clientStore, encodedData)
	if err != nil {
		panic(err)
	}
}

func (c ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	const VerifyClientMessage = "update_state"
	inner := make(map[string]interface{})
	inner["client_state"] = c
	clientMsgConcrete := make(map[string]interface{})
	switch clientMsg := clientMsg.(type) {
	case *Header:
		clientMsgConcrete["header"] = clientMsg
	case *Misbehaviour:
		clientMsgConcrete["misbehaviour"] = clientMsg
	}
	inner["client_message"] = clientMsgConcrete
	payload := make(map[string]map[string]interface{})
	payload[VerifyClientMessage] = inner

	output, err := call[contractResult](payload, &c, ctx, clientStore)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(output.Data, &c); err != nil {
		panic(sdkerrors.Wrapf(ErrUnableToUnmarshalPayload, fmt.Sprintf("underlying error: %s", err.Error())))
	}

	setClientState(clientStore, cdc, &c)
	// TODO: do we need to set consensus state?
	// setConsensusState(clientStore, cdc, consensusState, header.GetHeight())
	// setConsensusMetadata(ctx, clientStore, header.GetHeight())

	return []exported.Height{c.LatestHeight}
}

func (c ClientState) CheckSubstituteAndUpdateState(
	ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore,
	substituteClientStore sdk.KVStore, substituteClient exported.ClientState,
) error {
	var (
		SubjectPrefix    = []byte("subject/")
		SubstitutePrefix = []byte("substitute/")
	)

	consensusState, err := GetConsensusState(subjectClientStore, cdc, c.LatestHeight)
	if err != nil {
		return sdkerrors.Wrapf(
			err, "unexpected error: could not get consensus state from clientstore at height: %d", c.GetLatestHeight(),
		)
	}

	store := NewWrappedStore(subjectClientStore, substituteClientStore, SubjectPrefix, SubstitutePrefix)

	const CheckSubstituteAndUpdateState = "check_substitute_and_update_state"
	payload := make(map[string]map[string]interface{})
	payload[CheckSubstituteAndUpdateState] = make(map[string]interface{})
	inner := payload[CheckSubstituteAndUpdateState]
	inner["client_state"] = c
	inner["subject_consensus_state"] = consensusState
	inner["substitute_client_state"] = substituteClient
	// inner["initial_height"] = initialHeight

	output, err := call[clientStateCallResponse](payload, &c, ctx, store)
	if err != nil {
		return err
	}

	output.resetImmutables(&c)
	return nil
}

func (c ClientState) VerifyUpgradeAndUpdateState(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	store sdk.KVStore,
	newClient exported.ClientState,
	newConsState exported.ConsensusState,
	proofUpgradeClient,
	proofUpgradeConsState []byte,
) error {
	wasmUpgradeConsState, ok := newConsState.(*ConsensusState)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "upgraded consensus state must be wasm light consensus state. expected %T, got: %T",
			&ConsensusState{}, wasmUpgradeConsState)
	}

	// last height of current counterparty chain must be client's latest height
	lastHeight := c.LatestHeight
	_, err := GetConsensusState(store, cdc, lastHeight)
	if err != nil {
		return sdkerrors.Wrap(err, "could not retrieve consensus state for lastHeight")
	}

	const checkForMisbehaviourMessage = "verify_upgrade_and_update_state_msg"
	payload := make(map[string]map[string]interface{})
	payload[checkForMisbehaviourMessage] = make(map[string]interface{})
	inner := payload[checkForMisbehaviourMessage]
	inner["old_client_state"] = c
	inner["upgrade_client_state"] = newClient
	inner["upgrade_consensus_state"] = newConsState
	inner["proof_upgrade_client"] = proofUpgradeClient
	inner["proof_upgrade_consensus_state"] = proofUpgradeConsState

	encodedData, err := json.Marshal(payload)
	if err != nil {
		return sdkerrors.Wrapf(ErrUnableToMarshalPayload, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	out, err := callContract(c.CodeId, ctx, store, encodedData)
	if err != nil {
		return sdkerrors.Wrapf(ErrUnableToCall, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	output := contractResult{}
	if err := json.Unmarshal(out.Data, &output); err != nil {
		return sdkerrors.Wrapf(ErrUnableToUnmarshalPayload, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	if !output.IsValid {
		return fmt.Errorf("%s error occurred while verifyig upgrade and updating client state", output.ErrorMsg)
	}

	return nil
}

// NewClientState creates a new ClientState instance.
func NewClientState(latestSequence uint64, consensusState *ConsensusState) *ClientState {
	return &ClientState{
		Data:         []byte{0},
		CodeId:       []byte{},
		LatestHeight: clienttypes.Height{},
	}
}

// / Calls the contract with the given payload and writes the result to `output`
func call[T ContractResult](payload any, c *ClientState, ctx sdk.Context, clientStore types.KVStore) (T, error) {
	var output T
	encodedData, err := json.Marshal(payload)
	if err != nil {
		return output, sdkerrors.Wrapf(ErrUnableToMarshalPayload, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	out, err := callContract(c.CodeId, ctx, clientStore, encodedData)
	if err != nil {
		return output, sdkerrors.Wrapf(ErrUnableToCall, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	if err := json.Unmarshal(out.Data, &output); err != nil {
		return output, sdkerrors.Wrapf(ErrUnableToUnmarshalPayload, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	if !output.Validate() {
		return output, fmt.Errorf("%s error occurred while calling contract", output.Error())
	}
	return output, nil
}
