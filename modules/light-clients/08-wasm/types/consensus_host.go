package types

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clientkeeper "github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// WasmTMConsensusHost implements the 02-client types.ConsensusHost interface.
type WasmTMConsensusHost struct {
	cdc codec.BinaryCodec
	tm  *clientkeeper.TendermintConsensusHost
}

var _ clienttypes.ConsensusHost = (*WasmTMConsensusHost)(nil)

// NewWasmTMConsensusHost creates and returns a new ConsensusHost for wasm tendermint consensus client state and consensus state self validation.
func NewWasmTMConsensusHost(cdc codec.BinaryCodec, tm *clientkeeper.TendermintConsensusHost) *WasmTMConsensusHost {
	return &WasmTMConsensusHost{
		cdc: cdc,
		tm:  tm,
	}
}

// GetSelfConsensusState implements the 02-client types.ConsensusHost interface.
func (w *WasmTMConsensusHost) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
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

// ValidateSelfClient implements the 02-client types.ConsensusHost interface.
func (w *WasmTMConsensusHost) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	wasmClientState, ok := clientState.(*ClientState)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "client must be a wasm client, expected: %T, got: %T", ClientState{}, wasmClientState)
	}

	if w == nil {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "wasm consensus host is nil")
	}

	if w.cdc == nil {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "wasm consensus host cdc is nil")
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
