package ica

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	store "github.com/cosmos/cosmos-sdk/store/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	modulev1 "github.com/cosmos/ibc-go/v6/api/ibc/applications/interchain_accounts/module/v1"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/client/cli"
	controllerkeeper "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/keeper"
	controllertypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/types"
	genesistypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/genesis/types"
	"github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host"
	"github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/keeper"

	hostkeeper "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/keeper"
	hosttypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/simulation"
	"github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	porttypes "github.com/cosmos/ibc-go/v6/modules/core/05-port/types"
	ibchost "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
)

var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}

	_ porttypes.IBCModule = host.IBCModule{}
)

// AppModuleBasic is the IBC interchain accounts AppModuleBasic
type AppModuleBasic struct{}

// Name implements AppModuleBasic interface
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec implements AppModuleBasic.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

// RegisterInterfaces registers module concrete types into protobuf Any
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	controllertypes.RegisterInterfaces(registry)
	types.RegisterInterfaces(registry)
}

// DefaultGenesis returns default genesis state as raw bytes for the IBC
// interchain accounts module
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(genesistypes.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the IBC interchain acounts module
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var gs genesistypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return gs.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the interchain accounts module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	err := controllertypes.RegisterQueryHandlerClient(context.Background(), mux, controllertypes.NewQueryClient(clientCtx))
	if err != nil {
		panic(err)
	}

	err = hosttypes.RegisterQueryHandlerClient(context.Background(), mux, hosttypes.NewQueryClient(clientCtx))
	if err != nil {
		panic(err)
	}
}

// GetTxCmd implements AppModuleBasic interface
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

// GetQueryCmd implements AppModuleBasic interface
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// AppModule is the application module for the IBC interchain accounts module
type AppModule struct {
	AppModuleBasic
	controllerKeeper *controllerkeeper.Keeper
	hostKeeper       *hostkeeper.Keeper
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// NewAppModule creates a new IBC interchain accounts module
func NewAppModule(controllerKeeper *controllerkeeper.Keeper, hostKeeper *hostkeeper.Keeper) AppModule {
	return AppModule{
		controllerKeeper: controllerKeeper,
		hostKeeper:       hostKeeper,
	}
}

// InitModule will initialize the interchain accounts moudule. It should only be
// called once and as an alternative to InitGenesis.
func (am AppModule) InitModule(ctx sdk.Context, controllerParams controllertypes.Params, hostParams hosttypes.Params) {
	if am.controllerKeeper != nil {
		am.controllerKeeper.SetParams(ctx, controllerParams)
	}

	if am.hostKeeper != nil {
		am.hostKeeper.SetParams(ctx, hostParams)

		cap := am.hostKeeper.BindPort(ctx, types.HostPortID)
		if err := am.hostKeeper.ClaimCapability(ctx, cap, ibchost.PortPath(types.HostPortID)); err != nil {
			panic(fmt.Sprintf("could not claim port capability: %v", err))
		}
	}
}

// RegisterInvariants implements the AppModule interface
func (AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
}

// RegisterServices registers module services
func (am AppModule) RegisterServices(cfg module.Configurator) {
	if am.controllerKeeper != nil {
		controllertypes.RegisterMsgServer(cfg.MsgServer(), controllerkeeper.NewMsgServerImpl(am.controllerKeeper))
		controllertypes.RegisterQueryServer(cfg.QueryServer(), am.controllerKeeper)
	}

	if am.hostKeeper != nil {
		hosttypes.RegisterQueryServer(cfg.QueryServer(), am.hostKeeper)
	}

	m := controllerkeeper.NewMigrator(am.controllerKeeper)
	if err := cfg.RegisterMigration(types.ModuleName, 1, m.AssertChannelCapabilityMigrations); err != nil {
		panic(fmt.Sprintf("failed to migrate interchainaccounts app from version 1 to 2: %v", err))
	}
}

// InitGenesis performs genesis initialization for the interchain accounts module.
// It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState genesistypes.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)

	if am.controllerKeeper != nil {
		controllerkeeper.InitGenesis(ctx, *am.controllerKeeper, genesisState.ControllerGenesisState)
	}

	if am.hostKeeper != nil {
		hostkeeper.InitGenesis(ctx, *am.hostKeeper, genesisState.HostGenesisState)
	}

	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the exported genesis state as raw bytes for the interchain accounts module
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
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

	return cdc.MustMarshalJSON(gs)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 2 }

// BeginBlock implements the AppModule interface
func (am AppModule) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {
}

// EndBlock implements the AppModule interface
func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

// AppModuleSimulation functions

// GenerateGenesisState creates a randomized GenState of the ics27 module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState)
}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// WeightedOperations is unimplemented.
func (am AppModule) WeightedOperations(_ module.SimulationState) []simtypes.WeightedOperation {
	return nil
}

// RandomizedParams creates randomized ibc-transfer param changes for the simulator.
func (am AppModule) RandomizedParams(r *rand.Rand) []simtypes.ParamChange {
	return simulation.ParamChanges(r, am.controllerKeeper, am.hostKeeper)
}

// RegisterStoreDecoder registers a decoder for interchain accounts module's types
func (am AppModule) RegisterStoreDecoder(sdr sdk.StoreDecoderRegistry) {
	sdr[types.StoreKey] = simulation.NewDecodeStore()
}

//
// App-Wiring Setup
//

func init() {
	appmodule.Register(&modulev1.Module{})
}

type InterchainAccountsInput struct {
	depinject.In

	Config *modulev1.Module
	Key    *store.KVStoreKey
	Cdc    codec.Codec

	controllerKeeper controllerkeeper.Keeper

	paramSpace    paramtypes.Subspace
	ics4Wrapper   porttypes.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	authKeeper    types.AccountKeeper
	scopedKeeper  exported.ScopedKeeper
	msgRouter     types.MessageRouter
}

type InterchainAccountsOutput struct {
	depinject.Out

	InterchainAccountsKeeper keeper.Keeper
	Module                   appmodule.AppModule
}

// ProvideModule is used to provide interchain accounts module as dependency
func ProvideModule(in InterchainAccountsInput) InterchainAccountsOutput {
	k := keeper.NewKeeper(in.Cdc, in.Key, in.paramSpace, in.ics4Wrapper, in.channelKeeper, in.portKeeper, in.authKeeper, in.scopedKeeper, in.msgRouter)
	m := NewAppModule(&in.controllerKeeper, &k)
	return InterchainAccountsOutput{InterchainAccountsKeeper: k, Module: m}
}
