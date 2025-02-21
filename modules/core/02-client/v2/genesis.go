package clientv2

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
)

// InitGenesis initializes the ibc client/v2 submodule's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k *keeper.Keeper, gs types.GenesisState) {
	if err := gs.Validate(); err != nil {
		panic(fmt.Errorf("invalid genesis state: %w", err))
	}

	for _, info := range gs.CounterpartyInfos {
		k.SetClientCounterparty(ctx, info.ClientId, info.CounterpartyInfo)
	}
}

// ExportGenesis returns the ibc client/v2 submodule's exported genesis.
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) types.GenesisState {
	clients := k.ClientV1Keeper.GetAllGenesisClients(ctx)
	gs := types.GenesisState{
		CounterpartyInfos: make([]types.GenesisCounterpartyInfo, 0),
	}
	for _, client := range clients {
		counterpartyInfo, found := k.GetClientCounterparty(ctx, client.ClientId)
		if found {
			gs.CounterpartyInfos = append(gs.CounterpartyInfos, types.GenesisCounterpartyInfo{
				ClientId:         client.ClientId,
				CounterpartyInfo: counterpartyInfo,
			})
		}
	}

	return gs
}
