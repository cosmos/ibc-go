package solomachine

import (
	"cosmossdk.io/core/appmodule"
	coreregistry "cosmossdk.io/core/registry"
)

var (
	_ appmodule.AppModule             = (*AppModule)(nil)
	_ appmodule.HasRegisterInterfaces = (*AppModule)(nil)
)

// AppModule is the application module for the Solomachine client module
type AppModule struct {
	lightClientModule LightClientModule
}

// NewAppModule creates a new Solomachine client module
func NewAppModule(lightClientModule LightClientModule) AppModule {
	return AppModule{
		lightClientModule: lightClientModule,
	}
}

// Name returns the solo machine module name.
func (AppModule) Name() string {
	return ModuleName
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// RegisterInterfaces registers module concrete types into protobuf Any. This allows core IBC
// to unmarshal solo machine types.
func (AppModule) RegisterInterfaces(registry coreregistry.InterfaceRegistrar) {
	RegisterInterfaces(registry)
}
