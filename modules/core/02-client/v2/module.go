package v2

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/v2/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/02-client/v2/types"
)

// Name returns the IBC channel ICS name.
func Name() string {
	return types.SubModuleName
}

// GetTxCmd returns the root tx command for IBC channels.
func GetTxCmd() *cobra.Command {
	return nil // TODO
}

// GetQueryCmd returns the root query command for IBC channels.
func GetQueryCmd() *cobra.Command {
	return nil // TODO
}

// InitGenesis initializes the ibc client/v2 submodule's state from a provided genesis
// state.
func InitGenesis(ctx context.Context, k *keeper.Keeper, gs types.GenesisState) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if err := gs.Validate(); err != nil {
		return fmt.Errorf("invalid genesis state: %w", err)
	}

	for _, counterparty := range gs.CounterpartyInfos {
		k.SetClientCounterparty(sdkCtx, counterparty.ClientId, counterparty)
	}

	return nil
}

// ExportGenesis returns the ibc client/v2 submodule's exported genesis.
func ExportGenesis(ctx context.Context, k *keeper.Keeper) types.GenesisState {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	clients := k.ClientV1Keeper.GetAllGenesisClients(ctx)
	gs := types.GenesisState{
		CounterpartyInfos: make([]types.CounterpartyInfo, 0),
	}
	for _, client := range clients {
		counterpartyInfo, found := k.GetClientCounterparty(sdkCtx, client.ClientId)
		if found {
			gs.CounterpartyInfos = append(gs.CounterpartyInfos, counterpartyInfo)
		}
	}

	return gs
}
