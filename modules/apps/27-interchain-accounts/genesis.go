package interchain_accounts

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/keeper"
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"

	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

// InitGenesis initializes the interchain accounts application state from a provided genesis state
func InitGenesis(ctx sdk.Context, keeper keeper.Keeper, state types.GenesisState) {
	if !keeper.IsBound(ctx, state.PortId) {
		cap := keeper.BindPort(ctx, state.PortId)
		if err := keeper.ClaimCapability(ctx, cap, host.PortPath(state.PortId)); err != nil {
			panic(fmt.Sprintf("could not claim port capability: %v", err))
		}
	}
}

// ExportGenesis exports transfer module's portID into its geneis state
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) *types.GenesisState {
	portID := keeper.GetPort(ctx)

	return &types.GenesisState{
		PortId: portID,
	}
}
