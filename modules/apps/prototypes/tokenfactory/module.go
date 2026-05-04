package tokenfactory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cosmos/ibc-go/v11/modules/apps/prototypes/tokenfactory/cli"
	"github.com/cosmos/ibc-go/v11/modules/apps/prototypes/tokenfactory/keeper"
	"github.com/cosmos/ibc-go/v11/modules/apps/prototypes/tokenfactory/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

var (
	_ module.AppModuleBasic      = (*AppModule)(nil)
	_ module.HasGenesis          = (*AppModule)(nil)
	_ module.HasConsensusVersion = (*AppModule)(nil)

	_ appmodule.AppModule       = (*AppModule)(nil)
	_ appmodule.HasBeginBlocker = (*AppModule)(nil)
	_ appmodule.HasEndBlocker   = (*AppModule)(nil)
)

const ConsensusVersion = 1

// AppModuleBasic implements the AppModuleBasic interface that defines the
// independent methods a Cosmos SDK module needs to implement.
type AppModuleBasic struct {
	cdc codec.BinaryCodec
}

func NewAppModuleBasic(cdc codec.BinaryCodec) AppModuleBasic {
	return AppModuleBasic{cdc: cdc}
}

// Name returns the name of the module as a string.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the amino codec for the module
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

// RegisterInterfaces registers the interfaces for the module
func (AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns the default genesis state
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return types.ValidateGenesis(genState)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// AppModule implements the AppModule interface that defines the inter-dependent methods that modules need to implement
type AppModule struct {
	AppModuleBasic

	keeper        keeper.Keeper
	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
}

// NewAppModule creates a new AppModule instance.
func NewAppModule(cdc codec.BinaryCodec, k keeper.Keeper, accountKeeper types.AccountKeeper, bankKeeper types.BankKeeper) AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(cdc),
		keeper:         k,
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
	}
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServer(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// InitGenesis performs the tokenfactory module's genesis initialization
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) {
	var genState types.GenesisState
	cdc.MustUnmarshalJSON(gs, &genState)

	am.keeper.InitGenesis(ctx, genState)
}

// ExportGenesis returns the tokenfactory module's exported genesis
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion implements ConsensusVersion
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// BeginBlock contains the logic that is automatically triggered at the beginning of each block
func (AppModule) BeginBlock(_ context.Context) error {
	return nil
}

// EndBlock contains the logic that is automatically triggered at the end of each block
func (AppModule) EndBlock(_ context.Context) error {
	return nil
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return types.AutoCLIOptions()
}

// GetTxCmd implements autocli.HasCustomTxCommand.
// Provides custom CLI commands for Mint and Burn which use cosmos.base.v1beta1.Coin
// positional args that are incompatible with AutoCLI's protov2 reflection.
func (AppModule) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}
