package keeper

import (
	"context"

	genesistypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/genesis/types"
)

// InitGenesis initializes the interchain accounts controller application state from a provided genesis state
func InitGenesis(ctx context.Context, keeper Keeper, state genesistypes.ControllerGenesisState) {
	for _, portID := range state.Ports {
		keeper.setPort(ctx, portID)
	}

	for _, ch := range state.ActiveChannels {
		keeper.SetActiveChannelID(ctx, ch.ConnectionId, ch.PortId, ch.ChannelId)

		if ch.IsMiddlewareEnabled {
			keeper.SetMiddlewareEnabled(ctx, ch.PortId, ch.ConnectionId)
		} else {
			keeper.SetMiddlewareDisabled(ctx, ch.PortId, ch.ConnectionId)
		}
	}

	for _, acc := range state.InterchainAccounts {
		keeper.SetInterchainAccountAddress(ctx, acc.ConnectionId, acc.PortId, acc.AccountAddress)
	}

	keeper.SetParams(ctx, state.Params)
}

// ExportGenesis returns the interchain accounts controller exported genesis
func ExportGenesis(ctx context.Context, keeper Keeper) genesistypes.ControllerGenesisState {
	return genesistypes.NewControllerGenesisState(
		keeper.GetAllActiveChannels(ctx),
		keeper.GetAllInterchainAccounts(ctx),
		keeper.GetAllPorts(ctx),
		keeper.GetParams(ctx),
	)
}
