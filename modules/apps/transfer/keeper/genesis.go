package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

// InitGenesis initializes the ibc-transfer state and binds to PortID.
func (k Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {
	k.SetPort(ctx, state.PortId)

	for _, denom := range state.Denoms {
		k.SetDenom(ctx, denom)
		k.SetDenomMetadata(ctx, denom)
	}

	k.SetParams(ctx, state.Params)

	// Every denom will have only one total escrow amount, since any
	// duplicate entry will fail validation in Validate of GenesisState
	for _, denomEscrow := range state.TotalEscrowed {
		k.SetTotalEscrowForDenom(ctx, denomEscrow)
	}
}

// ExportGenesis exports ibc-transfer module's portID and denom trace info into its genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		PortId:        k.GetPort(ctx),
		Denoms:        k.GetAllDenoms(ctx),
		Params:        k.GetParams(ctx),
		TotalEscrowed: k.GetAllTotalEscrowed(ctx),
	}
}
