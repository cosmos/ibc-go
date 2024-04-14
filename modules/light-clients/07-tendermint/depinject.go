package tendermint

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"

	modulev1 "github.com/cosmos/ibc-go/api/ibc/lightclients/tendermint/module/v1"
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

type ModuleInputs struct {
	depinject.In
}

type ModuleOutputs struct {
	depinject.Out

	Module appmodule.AppModule
}

// ProvideModule returns the 07-tendermint module outputs for dependency injection
func ProvideModule(in ModuleInputs) ModuleOutputs {
	m := NewAppModule()
	return ModuleOutputs{Module: m}
}
