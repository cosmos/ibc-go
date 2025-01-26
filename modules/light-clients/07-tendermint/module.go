package tendermint

import (
	"cosmossdk.io/core/appmodule"
	coreregistry "cosmossdk.io/core/registry"
)

var (
	_ appmodule.AppModule             = (*AppModule)(nil)
	_ appmodule.HasRegisterInterfaces = (*AppModule)(nil)
)

// AppModule is the application module for the Tendermint client module
type AppModule struct {
	lightClientModule LightClientModule
}

// NewAppModule creates a new Tendermint client module
func NewAppModule(lightClientModule LightClientModule) AppModule {
	return AppModule{
		lightClientModule: lightClientModule,
	}
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// Name returns the tendermint module name.
func (AppModule) Name() string {
	return ModuleName
}

// RegisterInterfaces registers module concrete types into protobuf Any. This allows core IBC
// to unmarshal tendermint light client types.
func (AppModule) RegisterInterfaces(registry coreregistry.InterfaceRegistrar) {
	RegisterInterfaces(registry)
}
