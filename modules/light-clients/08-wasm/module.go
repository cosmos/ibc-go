package wasm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	coreregistry "cosmossdk.io/core/registry"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/client/cli"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

var (
	_ appmodule.AppModule             = (*AppModule)(nil)
	_ appmodule.HasConsensusVersion   = (*AppModule)(nil)
	_ appmodule.HasRegisterInterfaces = (*AppModule)(nil)
	_ appmodule.HasMigrations         = (*AppModule)(nil)

	_ module.AppModule      = (*AppModule)(nil)
	_ module.HasGRPCGateway = (*AppModule)(nil)
	_ module.HasGenesis     = (*AppModule)(nil)

	_ autocli.HasCustomTxCommand    = (*AppModule)(nil)
	_ autocli.HasCustomQueryCommand = (*AppModule)(nil)
)

// AppModule represents the AppModule for this module
type AppModule struct {
	cdc    codec.Codec
	keeper keeper.Keeper
}

// NewAppModule creates a new 08-wasm module
func NewAppModule(cdc codec.Codec, k keeper.Keeper) AppModule {
	return AppModule{
		cdc:    cdc,
		keeper: k,
	}
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// Name returns the tendermint module name.
func (AppModule) Name() string {
	return types.ModuleName
}

// RegisterInterfaces registers module concrete types into protobuf Any. This allows core IBC
// to unmarshal Wasm light client types.
func (AppModule) RegisterInterfaces(reg coreregistry.InterfaceRegistrar) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns an empty state, i.e. no contracts
func (am AppModule) DefaultGenesis() json.RawMessage {
	return am.cdc.MustMarshalJSON(&types.GenesisState{
		Contracts: []types.Contract{},
	})
}

// ValidateGenesis performs a no-op.
func (am AppModule) ValidateGenesis(bz json.RawMessage) error {
	var gs types.GenesisState
	if err := am.cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return gs.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for Wasm client module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx))
	if err != nil {
		panic(err)
	}
}

// GetTxCmd implements AppModule interface
func (AppModule) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

// GetQueryCmd implements AppModule interface
func (AppModule) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg grpc.ServiceRegistrar) error {
	types.RegisterMsgServer(cfg, am.keeper)
	types.RegisterQueryServer(cfg, am.keeper)
	return nil
}

func (am AppModule) RegisterMigrations(registrar appmodule.MigrationRegistrar) error {
	wasmMigrator := keeper.NewMigrator(am.keeper)
	if err := registrar.Register(types.ModuleName, 1, wasmMigrator.MigrateChecksums); err != nil {
		return err
	}
	return nil
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 2 }

func (am AppModule) InitGenesis(ctx context.Context, bz json.RawMessage) error {
	var gs types.GenesisState
	err := am.cdc.UnmarshalJSON(bz, &gs)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal %s genesis state: %s", am.Name(), err))
	}
	err = am.keeper.InitGenesis(ctx, gs)
	if err != nil {
		return err
	}
	return nil
}

func (am AppModule) ExportGenesis(ctx context.Context) (json.RawMessage, error) {
	gs := am.keeper.ExportGenesis(ctx)
	return am.cdc.MarshalJSON(&gs)
}
