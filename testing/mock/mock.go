package mock

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	abci "github.com/cometbft/cometbft/abci/types"

	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

const (
	ModuleName = "mock"

	MemStoreKey = "memory:mock"

	PortID = ModuleName

	Version = "mock-version"
)

var (
	MockAcknowledgement     = channeltypes.NewResultAcknowledgement([]byte("mock acknowledgement"))
	MockFailAcknowledgement = channeltypes.NewErrorAcknowledgement(errors.New("mock failed acknowledgement"))
	MockPacketData          = []byte("mock packet data")
	MockFailPacketData      = []byte("mock failed packet data")
	MockAsyncPacketData     = []byte("mock async packet data")
	UpgradeVersion          = fmt.Sprintf("%s-v2", Version)
	// MockApplicationCallbackError should be returned when an application callback should fail. It is possible to
	// test that this error was returned using ErrorIs.
	MockApplicationCallbackError error = &applicationCallbackError{}
)

var (
	TestKey   = []byte("test-key")
	TestValue = []byte("test-value")
)

var (
	_ module.AppModuleBasic = (*AppModuleBasic)(nil)
	_ appmodule.AppModule   = (*AppModule)(nil)

	_ porttypes.IBCModule = (*IBCModule)(nil)
)

// AppModuleBasic is the mock AppModuleBasic.
type AppModuleBasic struct{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModuleBasic) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModuleBasic) IsAppModule() {}

// Name implements AppModuleBasic interface.
func (AppModuleBasic) Name() string {
	return ModuleName
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// RegisterLegacyAminoCodec implements AppModuleBasic interface.
func (AppModuleBasic) RegisterLegacyAminoCodec(*codec.LegacyAmino) {}

// RegisterInterfaces implements AppModuleBasic interface.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {}

// DefaultGenesis implements AppModuleBasic interface.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return nil
}

// ValidateGenesis implements the AppModuleBasic interface.
func (AppModuleBasic) ValidateGenesis(codec.JSONCodec, client.TxEncodingConfig, json.RawMessage) error {
	return nil
}

// RegisterGRPCGatewayRoutes implements AppModuleBasic interface.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

// GetTxCmd implements AppModuleBasic interface.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

// GetQueryCmd implements AppModuleBasic interface.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return nil
}

// AppModule represents the AppModule for the mock module.
type AppModule struct {
	AppModuleBasic
	ibcApps []*IBCApp
}

// NewAppModule returns a mock AppModule instance.
func NewAppModule() AppModule {
	return AppModule{}
}

// RegisterInvariants implements the AppModule interface.
func (AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {}

// RegisterServices implements the AppModule interface.
func (AppModule) RegisterServices(module.Configurator) {}

// InitGenesis implements the AppModule interface.
func (AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

// ExportGenesis implements the AppModule interface.
func (AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return nil
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 1 }

var _ exported.Path = KeyPath{}

// KeyPath defines a placeholder struct which implements the exported.Path interface
type KeyPath struct{}

// String implements the exported.Path interface
func (KeyPath) String() string {
	return ""
}

// Empty implements the exported.Path interface
func (KeyPath) Empty() bool {
	return false
}

var _ exported.Height = Height{}

// Height defines a placeholder struct which implements the exported.Height interface
type Height struct {
	exported.Height
}
