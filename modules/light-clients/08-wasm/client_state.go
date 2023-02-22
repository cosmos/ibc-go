package wasm

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
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

type ExportMetadataPayload struct {
	ExportMetadata ExportMetadataInnerPayload `json:"exportmetadata"`
}

type ExportMetadataInnerPayload struct {
	ClientState ClientState `json:"client_state"`
}

func (c ClientState) ExportMetadata(store sdk.KVStore) []exported.GenesisMetadata {
	payload := ExportMetadataPayload{
		ExportMetadata: ExportMetadataInnerPayload{
			ClientState: c,
		},
	}
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

type verifyMembershipPayloadInner struct {
	Height           exported.Height `json:"height"`
	DelayTimePeriod  uint64          `json:"delay_time_period"`
	DelayBlockPeriod uint64          `json:"delay_block_period"`
	Proof            []byte          `json:"proof"`
	Path             exported.Path   `json:"path"`
	Value            []byte          `json:"value"`
}

type verifyMembershipPayload struct {
	VerifyMembershipPayloadInner verifyMembershipPayloadInner `json:"verify_membership"`
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
	payload := verifyMembershipPayload{
		VerifyMembershipPayloadInner: verifyMembershipPayloadInner{
			Height:           height,
			DelayTimePeriod:  delayTimePeriod,
			DelayBlockPeriod: delayBlockPeriod,
			Proof:            proof,
			Path:             path,
			Value:            value,
		},
	}

	_, err := call[contractResult](payload, &c, ctx, clientStore)
	return err
}

type verifyNonMembershipPayload struct {
	VerifyNonMembershipPayloadInner verifyNonMembershipPayloadInner `json:"verify_non_membership"`
}
type verifyNonMembershipPayloadInner struct {
	Height           exported.Height `json:"height"`
	DelayTimePeriod  uint64          `json:"delay_time_period"`
	DelayBlockPeriod uint64          `json:"delay_block_period"`
	Proof            []byte          `json:"proof"`
	Path             exported.Path   `json:"path"`
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
	payload := verifyNonMembershipPayload{
		VerifyNonMembershipPayloadInner: verifyNonMembershipPayloadInner{
			Height:           height,
			DelayTimePeriod:  delayTimePeriod,
			DelayBlockPeriod: delayBlockPeriod,
			Proof:            proof,
			Path:             path,
		},
	}
	_, err := call[contractResult](payload, &c, ctx, clientStore)
	return err
}

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
	_, err := call[contractResult](payload, &c, ctx, clientStore)
	if err != nil {
		panic(err)
	}
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
	// TODO: do we need to set consensus state?
	// setConsensusState(clientStore, cdc, consensusState, header.GetHeight())
	// setConsensusMetadata(ctx, clientStore, header.GetHeight())

	return []exported.Height{c.LatestHeight}
}

type checkSubstituteAndUpdateStatePayload struct {
	CheckSubstituteAndUpdateState CheckSubstituteAndUpdateStatePayload `json:"check_substitute_and_update_state"`
}

type CheckSubstituteAndUpdateStatePayload struct {
	ClientState              ClientState             `json:"client_state"`
	SubjectConsensusState    exported.ConsensusState `json:"subject_consensus_state"`
	SubstituteConsensusState exported.ClientState    `json:"substitute_client_state"`
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

	payload := checkSubstituteAndUpdateStatePayload{
		CheckSubstituteAndUpdateState: CheckSubstituteAndUpdateStatePayload{
			ClientState:              c,
			SubjectConsensusState:    consensusState,
			SubstituteConsensusState: substituteClient,
		},
	}

	output, err := call[clientStateCallResponse](payload, &c, ctx, store)
	if err != nil {
		return err
	}

	output.resetImmutables(&c)
	return nil
}

type verifyUpgradeAndUpdateStatePayload struct {
	VerifyUpgradeAndUpdateStateMsg verifyUpgradeAndUpdateStateMsgPayload `json:"verify_upgrade_and_update_state_msg"`
}

type verifyUpgradeAndUpdateStateMsgPayload struct {
	ClientState                ClientState             `json:"old_client_state"`
	SubjectConsensusState      exported.ClientState    `json:"upgrade_client_state"`
	UpgradeConsensusState      exported.ConsensusState `json:"upgrade_consensus_state"`
	ProofUpgradeClient         []byte                  `json:"proof_upgrade_client"`
	ProofUpgradeConsensusState []byte                  `json:"proof_upgrade_consensus_state"`
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

	payload := verifyUpgradeAndUpdateStatePayload{
		VerifyUpgradeAndUpdateStateMsg: verifyUpgradeAndUpdateStateMsgPayload{
			ClientState:                c,
			SubjectConsensusState:      newClient,
			UpgradeConsensusState:      newConsState,
			ProofUpgradeClient:         proofUpgradeClient,
			ProofUpgradeConsensusState: proofUpgradeConsState,
		},
	}

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
func call[T ContractResult](payload any, c *ClientState, ctx sdk.Context, clientStore sdk.KVStore) (T, error) {
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
