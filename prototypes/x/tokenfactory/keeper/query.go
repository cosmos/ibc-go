package keeper

import (
	"context"

	"github.com/cosmos/sandbox-ledger/x/tokenfactory/types"
)

var _ types.QueryServer = Keeper{}

// DenomAuthorityMetadata implements types.QueryServer.
func (k Keeper) DenomAuthorityMetadata(ctx context.Context, req *types.QueryDenomAuthorityMetadataRequest) (*types.QueryDenomAuthorityMetadataResponse, error) {
	authorityMetadata, err := k.GetAuthorityMetadata(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	return &types.QueryDenomAuthorityMetadataResponse{
		AuthorityMetadata: authorityMetadata,
	}, nil
}

// DenomsByCreator implements types.QueryServer.
func (k Keeper) DenomsByCreator(ctx context.Context, req *types.QueryDenomsByCreatorRequest) (*types.QueryDenomsByCreatorResponse, error) {
	denoms, err := k.GetDenomsFromCreator(ctx, req.Creator)
	if err != nil {
		return nil, err
	}

	return &types.QueryDenomsByCreatorResponse{
		Denoms: denoms,
	}, nil
}

// Params implements types.QueryServer.
func (k Keeper) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}
