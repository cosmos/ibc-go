package ibc

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
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	ibcclient "github.com/cosmos/ibc-go/v9/modules/core/02-client"
	clientkeeper "github.com/cosmos/ibc-go/v9/modules/core/02-client/keeper"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectionkeeper "github.com/cosmos/ibc-go/v9/modules/core/03-connection/keeper"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channelkeeper "github.com/cosmos/ibc-go/v9/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/client/cli"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/simulation"
	"github.com/cosmos/ibc-go/v9/modules/core/types"
)

var (
	_ appmodule.AppModule             = (*AppModule)(nil)
	_ appmodule.HasBeginBlocker       = (*AppModule)(nil)
	_ appmodule.HasConsensusVersion   = (*AppModule)(nil)
	_ appmodule.HasRegisterInterfaces = (*AppModule)(nil)
	_ appmodule.HasMigrations         = (*AppModule)(nil)

	_ module.AppModule      = (*AppModule)(nil)
	_ module.HasGRPCGateway = (*AppModule)(nil)
	_ module.HasGenesis     = (*AppModule)(nil)

	_ module.HasLegacyProposalMsgs = (*AppModule)(nil)
	_ module.AppModuleSimulation   = (*AppModule)(nil)

	_ autocli.HasCustomTxCommand    = (*AppModule)(nil)
	_ autocli.HasCustomQueryCommand = (*AppModule)(nil)
)

// AppModule implements an application module for the ibc module.
type AppModule struct {
	cdc    codec.Codec
	keeper *keeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, k *keeper.Keeper) AppModule {
	return AppModule{
		cdc:    cdc,
		keeper: k,
	}
}

// Name returns the ibc module's name.
func (AppModule) Name() string {
	return exported.ModuleName
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// DefaultGenesis returns default genesis state as raw bytes for the ibc
// module.
func (am AppModule) DefaultGenesis() json.RawMessage {
	return am.cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the ibc module.
func (am AppModule) ValidateGenesis(bz json.RawMessage) error {
	var gs types.GenesisState
	if err := am.cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", exported.ModuleName, err)
	}

	return gs.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the ibc module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	err := clienttypes.RegisterQueryHandlerClient(context.Background(), mux, clienttypes.NewQueryClient(clientCtx))
	if err != nil {
		panic(err)
	}
	err = connectiontypes.RegisterQueryHandlerClient(context.Background(), mux, connectiontypes.NewQueryClient(clientCtx))
	if err != nil {
		panic(err)
	}
	err = channeltypes.RegisterQueryHandlerClient(context.Background(), mux, channeltypes.NewQueryClient(clientCtx))
	if err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the ibc module.
func (AppModule) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns no root query command for the ibc module.
func (AppModule) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// RegisterInterfaces registers module concrete types into protobuf Any.
func (AppModule) RegisterInterfaces(registry coreregistry.InterfaceRegistrar) {
	types.RegisterInterfaces(registry)
}

func (am AppModule) RegisterMigrations(registrar appmodule.MigrationRegistrar) error {
	clientMigrator := clientkeeper.NewMigrator(am.keeper.ClientKeeper)
	if err := registrar.Register(exported.ModuleName, 2, clientMigrator.Migrate2to3); err != nil {
		return err
	}

	connectionMigrator := connectionkeeper.NewMigrator(am.keeper.ConnectionKeeper)
	if err := registrar.Register(exported.ModuleName, 3, connectionMigrator.Migrate3to4); err != nil {
		return err
	}

	if err := registrar.Register(exported.ModuleName, 4, func(ctx context.Context) error {
		if err := clientMigrator.MigrateParams(ctx); err != nil {
			return err
		}

		return connectionMigrator.MigrateParams(ctx)
	}); err != nil {
		return err
	}

	channelMigrator := channelkeeper.NewMigrator(am.keeper.ChannelKeeper)
	if err := registrar.Register(exported.ModuleName, 5, channelMigrator.MigrateParams); err != nil {
		return err
	}

	if err := registrar.Register(exported.ModuleName, 6, clientMigrator.MigrateToStatelessLocalhost); err != nil {
		return err
	}

	return nil
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg grpc.ServiceRegistrar) error {
	clienttypes.RegisterMsgServer(cfg, am.keeper)
	connectiontypes.RegisterMsgServer(cfg, am.keeper)
	channeltypes.RegisterMsgServer(cfg, am.keeper)
	clienttypes.RegisterQueryServer(cfg, clientkeeper.NewQueryServer(am.keeper.ClientKeeper))
	connectiontypes.RegisterQueryServer(cfg, connectionkeeper.NewQueryServer(am.keeper.ConnectionKeeper))
	channeltypes.RegisterQueryServer(cfg, channelkeeper.NewQueryServer(am.keeper.ChannelKeeper))
	return nil
}

// InitGenesis performs genesis initialization for the ibc module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx context.Context, bz json.RawMessage) error {
	var gs types.GenesisState
	err := am.cdc.UnmarshalJSON(bz, &gs)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal %s genesis state: %s", exported.ModuleName, err))
	}
	return InitGenesis(ctx, *am.keeper, &gs)
}

// ExportGenesis returns the exported genesis state as raw bytes for the ibc
// module.
func (am AppModule) ExportGenesis(ctx context.Context) (json.RawMessage, error) {
	gs, err := ExportGenesis(ctx, *am.keeper)
	if err != nil {
		return nil, err
	}
	return am.cdc.MarshalJSON(gs)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 7 }

// BeginBlock returns the begin blocker for the ibc module.
func (am AppModule) BeginBlock(ctx context.Context) error {
	ibcclient.BeginBlocker(ctx, am.keeper.ClientKeeper)
	return nil
}

// AppModuleSimulation functions

// GenerateGenesisState creates a randomized GenState of the ibc module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState)
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return simulation.ProposalMsgs()
}

// RegisterStoreDecoder registers a decoder for ibc module's types
func (am AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {
	sdr[exported.StoreKey] = simulation.NewDecodeStore(*am.keeper)
}
