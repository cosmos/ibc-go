package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

// InitGenesis initializes the interchain accounts controller application state from a provided genesis state
func InitGenesis(ctx sdk.Context, keeper Keeper, state icatypes.ControllerGenesisState) {
	for _, portID := range state.Ports {
		if !keeper.IsBound(ctx, portID) {
			cap := keeper.icaKeeper.BindPort(ctx, types.ModuleName, portID)
			if err := keeper.ClaimCapability(ctx, cap, host.PortPath(portID)); err != nil {
				panic(fmt.Sprintf("could not claim port capability: %v", err))
			}
		}
	}

	for _, ch := range state.ActiveChannels {
		keeper.icaKeeper.SetActiveChannelID(ctx, types.ModuleName, ch.PortId, ch.ChannelId)
	}

	for _, acc := range state.InterchainAccounts {
		keeper.icaKeeper.SetInterchainAccountAddress(ctx, types.ModuleName, acc.PortId, acc.AccountAddress)
	}
}

// ExportGenesis returns the interchain accounts controller exported genesis
func ExportGenesis(ctx sdk.Context, keeper Keeper) icatypes.ControllerGenesisState {
	return icatypes.NewControllerGenesisState(
		keeper.icaKeeper.GetAllActiveChannels(ctx, types.ModuleName),
		keeper.icaKeeper.GetAllInterchainAccounts(ctx, types.ModuleName),
		keeper.icaKeeper.GetAllPorts(ctx, types.ModuleName),
	)
}
