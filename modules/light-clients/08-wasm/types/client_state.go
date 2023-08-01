package types

import (
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
func (ClientState) ClientType() string {
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
	payload := queryMsg{Status: &statusMsg{}}

	result, err := wasmQuery[statusResult](ctx, clientStore, &cs, payload)
	if err != nil {
		return exported.Unknown
	}

	return result.Status
}

// ZeroCustomFields returns a ClientState that is a copy of the current ClientState
// with all client customizable fields zeroed out
func (cs ClientState) ZeroCustomFields() exported.ClientState {
	return &cs
}

// GetTimestampAtHeight returns the timestamp in nanoseconds of the consensus state at the given height.
func (cs ClientState) GetTimestampAtHeight(
	ctx sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
) (uint64, error) {
	payload := queryMsg{
		TimestampAtHeight: &timestampAtHeightMsg{
			Height: height,
		},
	}

	result, err := wasmQuery[timestampAtHeightResult](ctx, clientStore, &cs, payload)
	if err != nil {
		return 0, errorsmod.Wrapf(err, "height (%s)", height)
	}

	return result.Timestamp, nil
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

	// The global store key can be used here to implement #4085
	// wasmStore := ctx.KVStore(WasmStoreKey)

	return wasmInit(ctx, clientStore, &cs, payload)
}

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

	payload := queryMsg{
		VerifyMembership: &verifyMembershipMsg{
			Height:           height,
			DelayTimePeriod:  delayTimePeriod,
			DelayBlockPeriod: delayBlockPeriod,
			Proof:            proof,
			Path:             path,
			Value:            value,
		},
	}
	_, err := wasmQuery[contractResult](ctx, clientStore, &cs, payload)
	return err
}

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

	payload := queryMsg{
		VerifyNonMembership: &verifyNonMembershipMsg{
			Height:           height,
			DelayTimePeriod:  delayTimePeriod,
			DelayBlockPeriod: delayBlockPeriod,
			Proof:            proof,
			Path:             path,
		},
	}
	_, err := wasmQuery[contractResult](ctx, clientStore, &cs, payload)
	return err
}

// GetCodeHashes returns all the code hashes stored.
func GetCodeHashes(ctx sdk.Context, cdc codec.BinaryCodec) []string {
	store := ctx.KVStore(WasmStoreKey)
	bz := store.Get([]byte(KeyCodeHashes))
	if len(bz) == 0 {
		return []string{}
	}
	var hashes CodeHashes
	cdc.MustUnmarshal(bz, &hashes)
	return hashes.CodeHashes
}

// AddCodeHash adds a new code hash to the list of stored code hashes.
func AddCodeHash(ctx sdk.Context, cdc codec.BinaryCodec, codeHash []byte) {
	hashes := GetCodeHashes(ctx, cdc)
	hashes = append(hashes, string(codeHash))

	store := ctx.KVStore(WasmStoreKey)
	bz := cdc.MustMarshal(&CodeHashes{CodeHashes: hashes})
	store.Set([]byte(KeyCodeHashes), bz)
}

// HasCodeHash returns true if the given code hash exists in the store and
// false otherwise.
func HasCodeHash(ctx sdk.Context, cdc codec.BinaryCodec, codeHash []byte) bool {
	hashes := GetCodeHashes(ctx, cdc)

	for _, hash := range hashes {
		if hash == string(codeHash) {
			return true
		}
	}
	return false
}
