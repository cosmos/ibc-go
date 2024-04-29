package mock

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	storetypes "cosmossdk.io/store/types"

	modulev1 "github.com/cosmos/ibc-go/api/mock/module/v1"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
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
	IBCKeeper        *ibckeeper.Keeper
}

// ModuleOutputs defines the core module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	Module         appmodule.AppModule
	IBCModule      IBCModule
	IBCModuleRoute porttypes.IBCModuleRoute
	ScopedKeeper   ScopedMockKeeper
}

// ProvideModule defines a depinject provider function to supply the module dependencies and return its outputs.
func ProvideModule(in ModuleInputs) ModuleOutputs {
	scopedKeeper := in.CapabilityKeeper.ScopeToModule(ModuleName)
	m := NewAppModule(in.IBCKeeper.PortKeeper)
	ibcModule := NewIBCModule(&m, NewIBCApp(ModuleName, scopedKeeper))

	return ModuleOutputs{
		Module:         m,
		IBCModule:      ibcModule,
		IBCModuleRoute: porttypes.IBCModuleRoute{Name: ModuleName, IBCModule: ibcModule},
		ScopedKeeper:   ScopedMockKeeper{ScopedKeeper: scopedKeeper},
	}
}
