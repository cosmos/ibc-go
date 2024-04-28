package tendermint

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"

	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	modulev1 "github.com/cosmos/ibc-go/api/ibc/lightclients/tendermint/module/v1"
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

// ModuleInputs defines the 07-tendermint module inputs for depinject.
type ModuleInputs struct {
	depinject.In

	Config *modulev1.Module
	Cdc    codec.Codec
}

// ModuleOutputs defines the 07-tendermint module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	LightClientModuleWrapper clienttypes.LightClientModuleWrapper
	Module                   appmodule.AppModule
}

// ProvideModule returns the 07-tendermint module outputs for dependency injection
func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	lightClientModule := NewLightClientModule(in.Cdc, authority.String())
	m := NewAppModule(lightClientModule)
	return ModuleOutputs{LightClientModuleWrapper: clienttypes.NewLightClientModuleWrapper(&lightClientModule), Module: m}
}
