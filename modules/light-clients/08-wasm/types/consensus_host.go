package types

import (
	"errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// WasmConsensusHost implements the 02-client types.ConsensusHost interface.
type WasmConsensusHost struct {
	cdc      codec.BinaryCodec
	delegate clienttypes.ConsensusHost
}

var _ clienttypes.ConsensusHost = (*WasmConsensusHost)(nil)

// NewWasmConsensusHost creates and returns a new ConsensusHost for wasm wrapped consensus client state.
func NewWasmConsensusHost(cdc codec.BinaryCodec, delegate clienttypes.ConsensusHost) (*WasmConsensusHost, error) {
	if cdc == nil {
		return nil, errors.New("wasm consensus host codec is nil")
	}

	if delegate == nil {
		return nil, errors.New("wasm delegate consensus host is nil")
	}

	return &WasmConsensusHost{
		cdc:      cdc,
		delegate: delegate,
	}, nil
}

// GetSelfConsensusState implements the 02-client types.ConsensusHost interface.
func (w *WasmConsensusHost) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
	consensusState, err := w.delegate.GetSelfConsensusState(ctx, height)
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
