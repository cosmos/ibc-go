package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
)

// InitGenesis initializes the ibc nft-transfer state and binds to PortID.
func (k Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {
	k.SetPort(ctx, state.PortId)

	for _, trace := range state.Traces {
		k.SetClassTrace(ctx, trace)
	}

	// Only try to bind to port if it is not already bound, since we may already own
	// port capability from capability InitGenesis
	if !k.IsBound(ctx, state.PortId) {
		// nft-transfer module binds to the nft-transfer port on InitChain
		// and claims the returned capability
		err := k.BindPort(ctx, state.PortId)
		if err != nil {
			panic(fmt.Sprintf("could not claim port capability: %v", err))
		}
	}
}

// ExportGenesis exports ibc nft-transfer  module's portID and class trace info into its genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		PortId: k.GetPort(ctx),
		Traces: k.GetAllClassTraces(ctx),
	}
}
