package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/client/cli"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

var (
	_ appmodule.AppModule = (*AppModule)(nil)

	_ module.AppModule  = (*AppModule)(nil)
	_ module.HasGenesis = (*AppModule)(nil)
)

// AppModule represents the AppModule for this module
type AppModule struct {
	cdc    codec.Codec
	keeper *keeper.Keeper
}

func (AppModule) IsAppModule() {}

func (AppModule) IsOnePerModuleType() {}

func (m AppModule) DefaultGenesis() json.RawMessage {
	gs := types.DefaultGenesisState()
	return m.cdc.MustMarshalJSON(&gs)
}

func (m AppModule) ValidateGenesis(data json.RawMessage) error {
	gs := &types.GenesisState{}
	err := m.cdc.UnmarshalJSON(data, gs)
	if err != nil {
		return err
	}

	return gs.Validate()
}

func (m AppModule) InitGenesis(ctx context.Context, data json.RawMessage) error {
	gs := &types.GenesisState{}
	err := m.cdc.UnmarshalJSON(data, gs)
	if err != nil {
		return fmt.Errorf("failed to unmarshal genesis state: %w", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	m.keeper.InitGenesis(sdkCtx, *gs)

	return nil
}

func (m AppModule) ExportGenesis(ctx context.Context) (json.RawMessage, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	gs := m.keeper.ExportGenesis(sdkCtx)
	return json.Marshal(gs)
}

// Name returns the IBC channel/v2 name
func (AppModule) Name() string {
	return types.SubModuleName
}

func Name() string {
	return AppModule{}.Name()
}

// GetQueryCmd returns the root query command for IBC channels v2.
func (AppModule) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// GetQueryCmd returns the root query command for IBC channels v2.
func GetQueryCmd() *cobra.Command {
	return AppModule{}.GetQueryCmd()
}

// GetTxCmd returns the root tx command for IBC channels v2.
func (AppModule) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

// GetTxCmd returns the root tx command for IBC channels v2.
func GetTxCmd() *cobra.Command {
	return AppModule{}.GetTxCmd()
}
