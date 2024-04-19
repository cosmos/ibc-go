package capability

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"

	modulev1 "github.com/cosmos/ibc-go/api/capability/module/v1"
	"github.com/cosmos/ibc-go/modules/capability/keeper"
)

var _ depinject.OnePerModuleType = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

func init() {
	appmodule.Register(
		&modulev1.Module{},
		appmodule.Provide(ProvideModule),
	)
}

// ModuleInputs defines the capability module inputs for depinject.
type ModuleInputs struct {
	depinject.In

	Config      *modulev1.Module
	Cdc         codec.Codec
	Key         *storetypes.KVStoreKey
	MemStoreKey *storetypes.MemoryStoreKey
}

// ModuleOutputs defines the capability module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	CapabilityKeeper *keeper.Keeper
	Module           appmodule.AppModule
}

// ProvideModule returns the capability module outputs for dependency injection
func ProvideModule(in ModuleInputs) ModuleOutputs {
	capabilityKeeper := keeper.NewKeeper(
		in.Cdc,
		in.Key,
		in.MemStoreKey,
	)
	m := NewAppModule(in.Cdc, *capabilityKeeper, in.Config.SealKeeper)

	return ModuleOutputs{CapabilityKeeper: capabilityKeeper, Module: m}
}
