package keeper

import (
	"context"
	"fmt"

	genesistypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
)

// InitGenesis initializes the interchain accounts host application state from a provided genesis state
func InitGenesis(ctx context.Context, keeper Keeper, state genesistypes.HostGenesisState) error {
	keeper.setPort(ctx, state.Port)

	for _, ch := range state.ActiveChannels {
		keeper.SetActiveChannelID(ctx, ch.ConnectionId, ch.PortId, ch.ChannelId)
	}

	for _, acc := range state.InterchainAccounts {
		keeper.SetInterchainAccountAddress(ctx, acc.ConnectionId, acc.PortId, acc.AccountAddress)
	}

	if err := state.Params.Validate(); err != nil {
		return fmt.Errorf("could not set ica host params at genesis: %w", err)
	}
	keeper.SetParams(ctx, state.Params)
	return nil
}

// ExportGenesis returns the interchain accounts host exported genesis
func ExportGenesis(ctx context.Context, keeper Keeper) genesistypes.HostGenesisState {
	return genesistypes.NewHostGenesisState(
		keeper.GetAllActiveChannels(ctx),
		keeper.GetAllInterchainAccounts(ctx),
		icatypes.HostPortID,
		keeper.GetParams(ctx),
	)
}
