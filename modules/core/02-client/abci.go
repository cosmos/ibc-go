package client

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"
)

// BeginBlocker is used to perform IBC client upgrades
func BeginBlocker(ctx sdk.Context, k *keeper.Keeper) {
	k.ClientUpgrades(ctx)
}
