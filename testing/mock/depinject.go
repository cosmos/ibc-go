package mock

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	storetypes "cosmossdk.io/store/types"

	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"

	modulev1 "github.com/cosmos/ibc-go/api/mock/module/v1"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
)

func init() {
	appmodule.Register(
		&modulev1.Module{},
		appmodule.Provide(ProvideModule),
	)
}

var _ depinject.OnePerModuleType = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// ModuleInputs defines the core module inputs for depinject.
type ModuleInputs struct {
	depinject.In

	Config *modulev1.Module

	// TODO: mock memKey is not directly used by the mock IBCModule but is included in app_v1 and used in some testing.
	// Verify what tests are using this and if it can be removed here. It is certainly used ibccallbacks testing with mock contract keeper.
	Key *storetypes.MemoryStoreKey

	CapabilityKeeper *capabilitykeeper.Keeper
	PortKeeper       PortKeeper
}

// ModuleOutputs defines the core module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	Module       appmodule.AppModule
	IBCModule    porttypes.IBCModuleRoute
	ScopedKeeper capabilitykeeper.ScopedKeeper
}

// ProvideModule defines a depinject provider function to supply the module dependencies and return its outputs.
func ProvideModule(in ModuleInputs) ModuleOutputs {
	scopedKeeper := in.CapabilityKeeper.ScopeToModule(ModuleName)
	m := NewAppModule(in.PortKeeper)
	ibcModule := NewIBCModule(&m, NewIBCApp(ModuleName, scopedKeeper))

	return ModuleOutputs{Module: m, IBCModule: porttypes.IBCModuleRoute{Name: ModuleName, IBCModule: ibcModule}, ScopedKeeper: scopedKeeper}
}
