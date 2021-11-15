package controller

import (
	"encoding/json"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/controller/keeper"
	controllertypes "github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/controller/types"
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	porttypes "github.com/cosmos/ibc-go/v2/modules/core/05-port/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ porttypes.IBCModule   = IBCModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string {
	return controllertypes.ModuleName
}

// RegisterLegacyAminoCodec implements AppModuleBasic interface
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultControllerGenesis())
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	return nil // TODO: Implement GenesisState validation
}

func (AppModuleBasic) RegisterRESTRoutes(ctx client.Context, rtr *mux.Router) {
}

func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return nil
}

// RegisterInterfaces registers module concrete types into protobuf Any.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the interchain accounts module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
}

type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

// NewAppModule creates an interchain accounts app module.
func NewAppModule(k keeper.Keeper) AppModule {
	return AppModule{
		keeper: k,
	}
}

func (AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
}

func (AppModule) Route() sdk.Route {
	return sdk.NewRoute(controllertypes.RouterKey, nil)
}

func (AppModule) NewHandler() sdk.Handler {
	return nil
}

func (AppModule) QuerierRoute() string {
	return controllertypes.QuerierRoute
}

// LegacyQuerierHandler implements the AppModule interface
func (am AppModule) LegacyQuerierHandler(legacyQuerierCdc *codec.LegacyAmino) sdk.Querier {
	return nil
}

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState types.ControllerGenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)

	keeper.InitGenesis(ctx, am.keeper, genesisState)

	return []abci.ValidatorUpdate{}
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := keeper.ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(gs)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock implements the AppModule interface
func (am AppModule) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {
}

// EndBlock implements the AppModule interface
func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}
