package wasm

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	modulev1 "github.com/cosmos/ibc-go/api/ibc/lightclients/wasm/module/v1"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
	wasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
	wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
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

// ModuleInputs defines the 08-wasm module inputs for depinject.
type ModuleInputs struct {
	depinject.In

	Config       *modulev1.Module
	Cdc          codec.Codec
	StoreService store.KVStoreService
	ClientKeeper wasmtypes.ClientKeeper

	QueryRouter ibcwasm.QueryRouter
	Opts        []wasmkeeper.Option

	VM       ibcwasm.WasmEngine   `optional:"true"`
	VMConfig wasmtypes.WasmConfig `optional:"true"`
}

// ModuleOutputs defines the 08-wasm module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	LightClientModule *LightClientModule
	Module            appmodule.AppModule
}

// ProvideModule returns the 08-wasm module outputs for dependency injection
func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	var keeper wasmkeeper.Keeper
	if in.VM != nil {
		keeper = wasmkeeper.NewKeeperWithVM(
			in.Cdc, in.StoreService, in.ClientKeeper, authority.String(), in.VM, in.QueryRouter, in.Opts...,
		)
	} else {
		// TODO(jim): If missing, its default value is used. This could very well be surprising and cause misconfiguration
		keeper = wasmkeeper.NewKeeperWithConfig(
			in.Cdc, in.StoreService, in.ClientKeeper, authority.String(), in.VMConfig, in.QueryRouter, in.Opts...,
		)
	}

	lightClientModule := NewLightClientModule(keeper)
	m := NewAppModule(lightClientModule)
	return ModuleOutputs{LightClientModule: &lightClientModule, Module: m}
}
