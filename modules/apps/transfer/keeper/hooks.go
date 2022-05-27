package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
)

// AfterEpochEnd executes the indicated hook after epochs ends
func (k Keeper) AfterTransferEnd(ctx sdk.Context, packet types.FungibleTokenPacketData, base_denom string) {
	k.hooks.AfterTransferEnd(ctx, packet, base_denom)
}
