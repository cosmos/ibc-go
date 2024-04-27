package solomachine

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"

	"github.com/cosmos/cosmos-sdk/codec"

	modulev1 "github.com/cosmos/ibc-go/api/ibc/lightclients/solomachine/module/v1"
)

var (
	_ depinject.OnePerModuleType = AppModule{}
	_ depinject.OnePerModuleType = LightClientModule{}
)

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (LightClientModule) IsOnePerModuleType() {}

func init() {
	appmodule.Register(
		&modulev1.Module{},
		appmodule.Provide(ProvideModule),
	)
}

// ModuleInputs defines the 06-solomachine module inputs for depinject.
type ModuleInputs struct {
	depinject.In

	Cdc codec.Codec
}

// ModuleOutputs defines the 06-solomachine module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	LightClientModule *LightClientModule
	Module            appmodule.AppModule
}

// ProvideModule returns the 06-solomachine module outputs for dependency injection
func ProvideModule(in ModuleInputs) ModuleOutputs {
	lightClientModule := NewLightClientModule(in.Cdc)
	m := NewAppModule(lightClientModule)
	return ModuleOutputs{LightClientModule: &lightClientModule, Module: m}
}
