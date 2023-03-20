package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// HasCapability checks if the IBC app module owns the port capability for the desired port
func (k Keeper) HasCapability(ctx sdk.Context, portID string) bool {
	return k.hasCapability(ctx, portID)
}
