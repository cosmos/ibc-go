package mock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"
	coreregistry "cosmossdk.io/core/registry"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	feetypes "github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
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
	MockFeeVersion                     = string(feetypes.ModuleCdc.MustMarshalJSON(&feetypes.Metadata{FeeVersion: feetypes.Version, AppVersion: Version}))
)

var (
	TestKey   = []byte("test-key")
	TestValue = []byte("test-value")
)

var (
	_ appmodule.AppModule = (*AppModule)(nil)

	_ porttypes.IBCModule = (*IBCModule)(nil)
)

// AppModule represents the AppModule for the mock module.
type AppModule struct {
	ibcApps []*IBCApp
}

// NewAppModule returns a mock AppModule instance.
func NewAppModule() AppModule {
	return AppModule{}
}

// Name implements AppModule interface.
func (AppModule) Name() string {
	return ModuleName
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// RegisterLegacyAminoCodec implements AppModuleBasic interface.
func (AppModule) RegisterLegacyAminoCodec(coreregistry.AminoRegistrar) {}

// RegisterInterfaces implements AppModuleBasic interface.
func (AppModule) RegisterInterfaces(registry coreregistry.InterfaceRegistrar) {}

// DefaultGenesis implements AppModuleBasic interface.
func (AppModule) DefaultGenesis() json.RawMessage {
	return nil
}

// ValidateGenesis implements the AppModuleBasic interface.
func (AppModule) ValidateGenesis(json.RawMessage) error {
	return nil
}

// RegisterGRPCGatewayRoutes implements AppModuleBasic interface.
func (AppModule) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

// GetTxCmd implements AppModuleBasic interface.
func (AppModule) GetTxCmd() *cobra.Command {
	return nil
}

// GetQueryCmd implements AppModuleBasic interface.
func (AppModule) GetQueryCmd() *cobra.Command {
	return nil
}

// RegisterInvariants implements the AppModule interface.
func (AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {}

// RegisterServices implements the AppModule interface.
func (AppModule) RegisterServices(module.Configurator) {}

// InitGenesis implements the AppModule interface.
func (AppModule) InitGenesis(ctx sdk.Context, data json.RawMessage) error {
	return nil
}

// ExportGenesis implements the AppModule interface.
func (AppModule) ExportGenesis(ctx context.Context) (json.RawMessage, error) {
	return nil, nil
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
