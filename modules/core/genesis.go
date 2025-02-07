package ibc

import (
	"context"

	client "github.com/cosmos/ibc-go/v9/modules/core/02-client"
	clientv2 "github.com/cosmos/ibc-go/v9/modules/core/02-client/v2"
	connection "github.com/cosmos/ibc-go/v9/modules/core/03-connection"
	channel "github.com/cosmos/ibc-go/v9/modules/core/04-channel"
	"github.com/cosmos/ibc-go/v9/modules/core/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/types"
)

// InitGenesis initializes the ibc state from a provided genesis
// state.
func InitGenesis(ctx context.Context, k keeper.Keeper, gs *types.GenesisState) error {
	if err := client.InitGenesis(ctx, k.ClientKeeper, gs.ClientGenesis); err != nil {
		return err
	}
	connection.InitGenesis(ctx, k.ConnectionKeeper, gs.ConnectionGenesis)
	channel.InitGenesis(ctx, k.ChannelKeeper, gs.ChannelGenesis)

	return clientv2.InitGenesis(ctx, k.ClientV2Keeper, gs.ClientV2Genesis)
}

// ExportGenesis returns the ibc exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) (*types.GenesisState, error) {
	clientGenState, err := client.ExportGenesis(ctx, k.ClientKeeper)
	if err != nil {
		return nil, err
	}

	return &types.GenesisState{
		ClientGenesis:     clientGenState,
		ClientV2Genesis:   clientv2.ExportGenesis(ctx, k.ClientV2Keeper),
		ConnectionGenesis: connection.ExportGenesis(ctx, k.ConnectionKeeper),
		ChannelGenesis:    channel.ExportGenesis(ctx, k.ChannelKeeper),
	}, nil
}
