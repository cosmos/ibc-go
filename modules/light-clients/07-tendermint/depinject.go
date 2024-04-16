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

// ModuleInputs defines the 07-tendermint module inputs for depinject.
type ModuleInputs struct {
	depinject.In
}

// ModuleOutputs defines the 07-tendermint module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	Module appmodule.AppModule
}

// ProvideModule returns the 07-tendermint module outputs for dependency injection
func ProvideModule(in ModuleInputs) ModuleOutputs {
	m := NewAppModule()
	return ModuleOutputs{Module: m}
}
