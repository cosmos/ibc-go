package solomachine

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"

	"github.com/cosmos/cosmos-sdk/codec"

	modulev1 "github.com/cosmos/ibc-go/api/ibc/lightclients/solomachine/module/v1"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
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

// ModuleInputs defines the 06-solomachine module inputs for depinject.
type ModuleInputs struct {
	depinject.In

	Cdc codec.Codec
}

// ModuleOutputs defines the 06-solomachine module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	LightClientModuleWrapper clienttypes.LightClientModuleWrapper
	Module                   appmodule.AppModule
}

// ProvideModule returns the 06-solomachine module outputs for dependency injection
func ProvideModule(in ModuleInputs) ModuleOutputs {
	lightClientModule := NewLightClientModule(in.Cdc)
	m := NewAppModule(lightClientModule)
	return ModuleOutputs{LightClientModuleWrapper: clienttypes.NewLightClientModuleWrapper(&lightClientModule), Module: m}
}
