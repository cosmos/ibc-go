package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v11/modules/apps/prototypes/ift/types"

	"cosmossdk.io/collections"
)

// InitGenesis initializes the IFT module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) {
	// Default the params authority to the keeper's authority (gov module address)
	// when not explicitly set in genesis.
	if genState.Params.Authority == "" {
		genState.Params.Authority = k.authority
	}

	if err := k.ParamsStore.Set(ctx, genState.Params); err != nil {
		panic(err)
	}

	for _, genBridge := range genState.Bridges {
		if err := k.IFTBridgeStore.Set(ctx, collections.Join(genBridge.Denom, genBridge.Bridge.ClientId), genBridge.Bridge); err != nil {
			panic(err)
		}
	}

	for _, pending := range genState.PendingTransfers {
		if err := k.SetPendingTransfer(ctx, pending.ClientId, pending.Sequence, pending); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the IFT module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := k.ParamsStore.Get(ctx)
	if err != nil {
		panic(err)
	}

	var bridges []types.GenesisBridge
	if err := k.IFTBridgeStore.Walk(ctx, nil, func(key collections.Pair[string, string], bridge types.IFTBridge) (bool, error) {
		bridges = append(bridges, types.GenesisBridge{
			Denom:  key.K1(),
			Bridge: bridge,
		})
		return false, nil
	}); err != nil {
		panic(err)
	}

	var pendingTransfers []types.PendingTransfer
	if err := k.PendingTransferStore.Walk(ctx, nil, func(_ collections.Pair[string, uint64], pending types.PendingTransfer) (bool, error) {
		pendingTransfers = append(pendingTransfers, pending)
		return false, nil
	}); err != nil {
		panic(err)
	}

	return &types.GenesisState{
		Params:           params,
		Bridges:          bridges,
		PendingTransfers: pendingTransfers,
	}
}
