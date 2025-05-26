package ratelimiting

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/client/cli"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/keeper"
	ratesim "github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/simulation"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

var (
	_ module.AppModule           = (*AppModule)(nil)
	_ module.AppModuleBasic      = (*AppModuleBasic)(nil)
	_ module.HasGenesis          = (*AppModule)(nil)
	_ module.HasName             = (*AppModule)(nil)
	_ module.HasConsensusVersion = (*AppModule)(nil)
	_ module.HasServices         = (*AppModule)(nil)
	_ module.HasProposalMsgs     = (*AppModule)(nil)
	_ appmodule.AppModule        = (*AppModule)(nil)
	_ appmodule.HasBeginBlocker  = (*AppModule)(nil) // Add HasBeginBlocker

	// Note: IBCMiddleware implements porttypes.Middleware and porttypes.ICS4Wrapper
)

// AppModuleBasic is the rate-limiting AppModuleBasic
type AppModuleBasic struct{}

// Name implements AppModuleBasic interface
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// RegisterLegacyAminoCodec implements AppModuleBasic interface
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers module concrete types into protobuf Any.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// DefaultGenesis returns default genesis state as raw bytes for the rate-limiting
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the rate-limiting module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return gs.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the rate-limiting module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
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

// AppModule represents the AppModule for this module
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

// NewAppModule creates a new rate-limiting module
func NewAppModule(k keeper.Keeper) AppModule {
	return AppModule{
		keeper: k,
	}
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper)) // Use the msgServer implementation
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)

	m := keeper.NewMigrator(am.keeper)
	if err := cfg.RegisterMigration(types.ModuleName, 1, m.MigrateParams); err != nil {
		panic(fmt.Errorf("failed to migrate rate-limiting app from version 1 to 2 (self-managed params migration): %v", err))
	}
}

// InitGenesis performs genesis initialization for the rate-limiting module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	am.keeper.InitGenesis(ctx, genesisState)
}

// ExportGenesis returns the exported genesis state as raw bytes for the rate-limiting
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}

// ConsensusVersion implements AppModule/ConsensusVersion defining the current version of rate-limiting.
func (AppModule) ConsensusVersion() uint64 { return 2 }

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return ratesim.ProposalMsgs()
}

// RegisterStoreDecoder registers a decoder for rate-limiting module's types
func (AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {
	sdr[types.StoreKey] = ratesim.NewDecodeStore()
}

// WeightedOperations returns the all the rate-limiting module operations with their respective weights.
func (AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return []simtypes.WeightedOperation{}
}

// GenerateGenesisState creates a randomized GenState of the rate-limiting module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	ratesim.RandomizedGenState(simState)
}

// BeginBlock implements the AppModule interface
func (am AppModule) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	am.keeper.BeginBlocker(sdkCtx)
	// we do not want to raise an error in block processing if rate limit reset fails
	return nil
}
