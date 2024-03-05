package types

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clientkeeper "github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

type WasmTMClientValidator struct {
	cdc codec.BinaryCodec
	tm  *clientkeeper.TendermintClientValidator
}

var _ clienttypes.SelfClientValidator = (*WasmTMClientValidator)(nil)

// NewWasmTMClientValidator creates and returns a new SelfClientValidator for wasm tendermint consensus.
func NewWasmTMClientValidator(cdc codec.BinaryCodec, tm *clientkeeper.TendermintClientValidator) *WasmTMClientValidator {
	return &WasmTMClientValidator{
		cdc: cdc,
		tm:  tm,
	}
}

func (w *WasmTMClientValidator) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
	consensusState, err := w.tm.GetSelfConsensusState(ctx, height)
	if err != nil {
		return nil, err
	}

	// encode consensusState to wasm.ConsensusState.Data
	bz, err := w.cdc.MarshalInterface(consensusState)
	if err != nil {
		return nil, err
	}

	wasmConsensusState := &ConsensusState{
		Data: bz,
	}

	return wasmConsensusState, nil
}

func (w *WasmTMClientValidator) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	wasmClientState, ok := clientState.(*ClientState)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "client must be a wasm client, expected: %T, got: %T", ClientState{}, wasmClientState)
	}

	if w == nil {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "wasm client validator is nil")
	}

	if w.cdc == nil {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "wasm client validator cdc is nil")
	}

	if wasmClientState.Data == nil {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "wasm client state data is nil")
	}

	// unmarshal the wasmClientState bytes into tendermint client and call self validation
	var tmClientState exported.ClientState
	if err := w.cdc.UnmarshalInterface(wasmClientState.Data, &tmClientState); err != nil {
		return err
	}

	return w.tm.ValidateSelfClient(ctx, tmClientState)
}
