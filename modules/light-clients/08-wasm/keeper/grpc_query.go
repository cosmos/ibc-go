package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

var _ types.QueryServer = Keeper{}

// WasmCode implements the IBC QueryServer interface
func (q Keeper) WasmCode(c context.Context, req *types.WasmCodeQuery) (*types.WasmCodeResponse, error) {
	return q.getWasmCode(c, req)
}

// AllWasmCodeID implements the IBC QueryServer interface
func (q Keeper) AllWasmCodeID(c context.Context, req *types.AllWasmCodeIDQuery) (*types.AllWasmCodeIDResponse, error) {
	return q.getAllWasmCodeID(c, req)
}
