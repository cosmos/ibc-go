package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
)

// InitGenesis
func (k Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {
	for _, fee := range state.IdentifiedFees {
		k.SetFeeInEscrow(ctx, fee)
	}
}

// ExportGenesis
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		IdentifiedFees: k.GetAllIdentifiedPacketFees(ctx),
	}
}
