package types

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

type statusPayloadInner struct {}
type statusPayload struct {
	Status statusPayloadInner `json:"status"`
}
func (c ClientState) Status(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec) exported.Status {
	status := exported.Unknown
	payload := statusPayload{Status: statusPayloadInner{}}

	encodedData, err := json.Marshal(payload)
	if err != nil {
		return status
	}

	response, err := queryContractWithStore(c.CodeId, ctx, store, encodedData)
	if err != nil {
		return status
	}
	output := queryResponse{}
	if err := json.Unmarshal(response, &output); err != nil {
		return status
	}

	return output.Status
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

// NewClientState creates a new ClientState instance.
func NewClientState(data []byte, codeID []byte, height clienttypes.Height) *ClientState {
	return &ClientState{
		Data:         data,
		CodeId:       codeID,
		LatestHeight: height,
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
