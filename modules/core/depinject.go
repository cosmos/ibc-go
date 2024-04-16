package ibc

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	modulev1 "github.com/cosmos/ibc-go/api/ibc/core/module/v1"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
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

// ModuleInputs defines the core module inputs for depinject.
type ModuleInputs struct {
	depinject.In

	Config *modulev1.Module
	Cdc    codec.Codec
	Key    *storetypes.KVStoreKey

	StakingKeeper clienttypes.StakingKeeper
	UpgradeKeeper clienttypes.UpgradeKeeper
	ScopedKeeper  capabilitykeeper.ScopedKeeper

	// LegacySubspace is used solely for migration of x/params managed parameters
	LegacySubspace paramtypes.Subspace `optional:"true"`
}

// ModuleOutputs defines the core module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	IbcKeeper *ibckeeper.Keeper
	Module    appmodule.AppModule
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	keeper := ibckeeper.NewKeeper(
		in.Cdc,
		in.Key,
		in.LegacySubspace,
		in.StakingKeeper,
		in.UpgradeKeeper,
		in.ScopedKeeper,
		authority.String(),
	)
	m := NewAppModule(keeper)

	return ModuleOutputs{IbcKeeper: keeper, Module: m}
}
