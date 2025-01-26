package ica

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

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/client/cli"
	controllerkeeper "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/keeper"
	controllertypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	genesistypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/genesis/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host"
	hostkeeper "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/keeper"
	hosttypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/simulation"
	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
)

var (
	_ appmodule.AppModule             = (*AppModule)(nil)
	_ appmodule.HasConsensusVersion   = (*AppModule)(nil)
	_ appmodule.HasRegisterInterfaces = (*AppModule)(nil)
	_ appmodule.HasMigrations         = (*AppModule)(nil)

	_ module.AppModule      = (*AppModule)(nil)
	_ module.HasGenesis     = (*AppModule)(nil)
	_ module.HasGRPCGateway = (*AppModule)(nil)

	// Sims
	_ module.AppModuleSimulation   = (*AppModule)(nil)
	_ module.HasLegacyProposalMsgs = (*AppModule)(nil)

	_ autocli.HasCustomTxCommand    = (*AppModule)(nil)
	_ autocli.HasCustomQueryCommand = (*AppModule)(nil)

	_ porttypes.IBCModule = (*host.IBCModule)(nil)
)

// AppModule is the application module for the IBC interchain accounts module
type AppModule struct {
	cdc              codec.Codec
	controllerKeeper *controllerkeeper.Keeper
	hostKeeper       *hostkeeper.Keeper
}

// NewAppModule creates a new IBC interchain accounts module
func NewAppModule(cdc codec.Codec, controllerKeeper *controllerkeeper.Keeper, hostKeeper *hostkeeper.Keeper) AppModule {
	return AppModule{
		cdc:              cdc,
		controllerKeeper: controllerKeeper,
		hostKeeper:       hostKeeper,
	}
}

// Name implements AppModuleBasic interface
func (AppModule) Name() string {
	return types.ModuleName
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// RegisterInterfaces registers module concrete types into protobuf Any
func (AppModule) RegisterInterfaces(registry coreregistry.InterfaceRegistrar) {
	controllertypes.RegisterInterfaces(registry)
	hosttypes.RegisterInterfaces(registry)
	types.RegisterInterfaces(registry)
}

// DefaultGenesis returns default genesis state as raw bytes for the IBC
// interchain accounts module
func (am AppModule) DefaultGenesis() json.RawMessage {
	return am.cdc.MustMarshalJSON(genesistypes.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the IBC interchain accounts module
func (am AppModule) ValidateGenesis(bz json.RawMessage) error {
	var gs genesistypes.GenesisState
	if err := am.cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return gs.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the interchain accounts module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	err := controllertypes.RegisterQueryHandlerClient(context.Background(), mux, controllertypes.NewQueryClient(clientCtx))
	if err != nil {
		panic(err)
	}

	err = hosttypes.RegisterQueryHandlerClient(context.Background(), mux, hosttypes.NewQueryClient(clientCtx))
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

func (am AppModule) RegisterMigrations(registrar appmodule.MigrationRegistrar) error {
	controllerMigrator := controllerkeeper.NewMigrator(am.controllerKeeper)
	hostMigrator := hostkeeper.NewMigrator(am.hostKeeper)
	if err := registrar.Register(types.ModuleName, 2, func(ctx context.Context) error {
		if err := hostMigrator.MigrateParams(ctx); err != nil {
			return err
		}
		return controllerMigrator.MigrateParams(ctx)
	}); err != nil {
		return fmt.Errorf("failed to migrate interchainaccounts app from version 2 to 3 (self-managed params migration): %w", err)
	}
	return nil
}

// RegisterServices registers module services
func (am AppModule) RegisterServices(cfg grpc.ServiceRegistrar) error {
	if am.controllerKeeper != nil {
		controllertypes.RegisterMsgServer(cfg, controllerkeeper.NewMsgServerImpl(am.controllerKeeper))
		controllertypes.RegisterQueryServer(cfg, am.controllerKeeper)
	}

	if am.hostKeeper != nil {
		hosttypes.RegisterMsgServer(cfg, hostkeeper.NewMsgServerImpl(am.hostKeeper))
		hosttypes.RegisterQueryServer(cfg, am.hostKeeper)
	}

	return nil
}

// InitGenesis performs genesis initialization for the interchain accounts module.
// It returns no validator updates.
func (am AppModule) InitGenesis(ctx context.Context, data json.RawMessage) error {
	var genesisState genesistypes.GenesisState
	if err := am.cdc.UnmarshalJSON(data, &genesisState); err != nil {
		return err
	}

	if am.controllerKeeper != nil {
		controllerkeeper.InitGenesis(ctx, *am.controllerKeeper, genesisState.ControllerGenesisState)
	}

	if am.hostKeeper != nil {
		hostkeeper.InitGenesis(ctx, *am.hostKeeper, genesisState.HostGenesisState)
	}
	return nil
}

// ExportGenesis returns the exported genesis state as raw bytes for the interchain accounts module
func (am AppModule) ExportGenesis(ctx context.Context) (json.RawMessage, error) {
	var (
		controllerGenesisState = genesistypes.DefaultControllerGenesis()
		hostGenesisState       = genesistypes.DefaultHostGenesis()
	)

	if am.controllerKeeper != nil {
		controllerGenesisState = controllerkeeper.ExportGenesis(ctx, *am.controllerKeeper)
	}

	if am.hostKeeper != nil {
		hostGenesisState = hostkeeper.ExportGenesis(ctx, *am.hostKeeper)
	}

	gs := genesistypes.NewGenesisState(controllerGenesisState, hostGenesisState)

	return am.cdc.MarshalJSON(gs)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 3 }

// AppModuleSimulation functions

// GenerateGenesisState creates a randomized GenState of the ics27 module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState)
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return simulation.ProposalMsgs(am.controllerKeeper, am.hostKeeper)
}

// RegisterStoreDecoder registers a decoder for interchain accounts module's types
func (AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {
	sdr[controllertypes.StoreKey] = simulation.NewDecodeStore()
	sdr[hosttypes.StoreKey] = simulation.NewDecodeStore()
}
