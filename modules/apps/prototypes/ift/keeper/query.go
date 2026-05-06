package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v11/modules/apps/prototypes/ift/types"

	"cosmossdk.io/collections"
)

var _ types.QueryServer = Keeper{}

// IFTBridge queries the IFT bridge information for a specific denom and client_id
func (k Keeper) IFTBridge(ctx context.Context, req *types.QueryIFTBridgeRequest) (*types.QueryIFTBridgeResponse, error) {
	bridge, err := k.IFTBridgeStore.Get(ctx, collections.Join(req.Denom, req.ClientId))
	if err != nil {
		return nil, err
	}

	return &types.QueryIFTBridgeResponse{Bridge: bridge}, nil
}

// IFTBridges queries all IFT bridges
func (k Keeper) IFTBridges(ctx context.Context, _ *types.QueryIFTBridgesRequest) (*types.QueryIFTBridgesResponse, error) {
	var bridges []types.DenomBridge

	err := k.IFTBridgeStore.Walk(ctx, nil, func(key collections.Pair[string, string], bridge types.IFTBridge) (bool, error) {
		bridges = append(bridges, types.DenomBridge{
			Denom:  key.K1(),
			Bridge: bridge,
		})
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryIFTBridgesResponse{Bridges: bridges}, nil
}

// IFTBridgesByDenom queries all IFT bridges for a specific denom
func (k Keeper) IFTBridgesByDenom(ctx context.Context, req *types.QueryIFTBridgesByDenomRequest) (*types.QueryIFTBridgesByDenomResponse, error) {
	var bridges []types.IFTBridge

	rng := collections.NewPrefixedPairRange[string, string](req.Denom)
	err := k.IFTBridgeStore.Walk(ctx, rng, func(_ collections.Pair[string, string], bridge types.IFTBridge) (bool, error) {
		bridges = append(bridges, bridge)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryIFTBridgesByDenomResponse{Bridges: bridges}, nil
}

// PendingTransfer queries a pending transfer
func (k Keeper) PendingTransfer(ctx context.Context, req *types.QueryPendingTransferRequest) (*types.QueryPendingTransferResponse, error) {
	pending, err := k.PendingTransferStore.Get(ctx, collections.Join(req.ClientId, req.Sequence))
	if err != nil {
		return nil, err
	}

	// Validate denom matches if specified
	if req.Denom != "" && pending.Denom != req.Denom {
		return nil, collections.ErrNotFound
	}

	return &types.QueryPendingTransferResponse{PendingTransfer: pending}, nil
}

// Params queries the module parameters
func (k Keeper) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := k.ParamsStore.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: params}, nil
}
