package client

import (
	"context"

	"github.com/spf13/cobra"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/client/cli"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

func InitGenesis(ctx context.Context, k *keeper.Keeper, gs types.GenesisState) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.InitGenesis(sdkCtx, gs)
}

func ExportGenesis(ctx context.Context, k *keeper.Keeper) types.GenesisState {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return k.ExportGenesis(sdkCtx)
}

func Name() string {
	return types.SubModuleName
}

// GetQueryCmd returns the root query command for IBC channels v2.
func GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// GetTxCmd returns the root tx command for IBC channels v2.
func GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}
