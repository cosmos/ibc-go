package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetParams sets the total set of params.
func (k Keeper) HasCapability(ctx sdk.Context, portID string) bool {
	return k.hasCapability(ctx, portID)
}
