package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the ratelimit MsgServer interface
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// Adds a new rate limit. Fails if the rate limit already exists or the channel value is 0
func (k msgServer) AddRateLimit(goCtx context.Context, msg *types.MsgAddRateLimit) (*types.MsgAddRateLimitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// if k.authority != msg.Signer {
	// 	return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Signer)
	// }

	if err := k.Keeper.AddRateLimit(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgAddRateLimitResponse{}, nil
}

// Updates an existing rate limit. Fails if the rate limit doesn't exist
func (k msgServer) UpdateRateLimit(goCtx context.Context, msg *types.MsgUpdateRateLimit) (*types.MsgUpdateRateLimitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// if k.authority != msg.Signer {
	// 	return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Signer)
	// }

	if err := k.Keeper.UpdateRateLimit(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgUpdateRateLimitResponse{}, nil
}

// Removes a rate limit. Fails if the rate limit doesn't exist
func (k msgServer) RemoveRateLimit(goCtx context.Context, msg *types.MsgRemoveRateLimit) (*types.MsgRemoveRateLimitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// if k.authority != msg.Signer {
	// 	return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Signer)
	// }

	_, found := k.GetRateLimit(ctx, msg.Denom, msg.ChannelOrClientId)
	if !found {
		return nil, types.ErrRateLimitNotFound
	}

	k.Keeper.RemoveRateLimit(ctx, msg.Denom, msg.ChannelOrClientId)
	return &types.MsgRemoveRateLimitResponse{}, nil
}

// Resets the flow on a rate limit. Fails if the rate limit doesn't exist
func (k msgServer) ResetRateLimit(goCtx context.Context, msg *types.MsgResetRateLimit) (*types.MsgResetRateLimitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// if k.authority != msg.Signer {
	// 	return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Signer)
	// }

	if err := k.Keeper.ResetRateLimit(ctx, msg.Denom, msg.ChannelOrClientId); err != nil {
		return nil, err
	}

	return &types.MsgResetRateLimitResponse{}, nil
}
