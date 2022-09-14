package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
)

// AfterEpochEnd executes the indicated hook after Transfer ends
func (k Keeper) AfterTransferEnd(ctx sdk.Context, packet types.FungibleTokenPacketData, base_denom string) {
	k.hooks.AfterTransferEnd(ctx, packet, base_denom)
}

// AfterOnRecvPacket executes the indicated hook after OnRecvPacket ends
func (k Keeper) AfterOnRecvPacket(ctx sdk.Context, packet types.FungibleTokenPacketData) {
	k.hooks.AfterOnRecvPacket(ctx, packet)
}
