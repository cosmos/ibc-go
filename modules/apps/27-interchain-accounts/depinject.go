package ica

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	modulev1 "github.com/cosmos/ibc-go/api/ibc/applications/interchain_accounts/module/v1"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	controllerKeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	hostKeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
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

// ModuleInputs defines the 27-interchain-accounts module inputs for depinject.
type ModuleInputs struct {
	depinject.In

	Config *modulev1.Module
	Cdc    codec.Codec
	Key    *storetypes.KVStoreKey

	Ics4Wrapper   porttypes.ICS4Wrapper
	ChannelKeeper types.ChannelKeeper
	PortKeeper    types.PortKeeper
	ScopedKeeper  capabilitykeeper.ScopedKeeper
	AccountKeeper types.AccountKeeper

	MsgRouter   types.MessageRouter
	QueryRouter types.QueryRouter

	// LegacySubspace is used solely for migration of x/params managed parameters
	LegacySubspace paramtypes.Subspace `optional:"true"`
}

// ModuleOutputs defines the 27-interchain-accounts module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	ControllerKeeper *controllerKeeper.Keeper
	HostKeeper       *hostKeeper.Keeper
	Module           appmodule.AppModule
}

// ProvideModule returns the 27-interchain-accounts outputs for dependency injection
func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	controllerkeeper := controllerKeeper.NewKeeper(
		in.Cdc,
		in.Key,
		in.LegacySubspace,
		in.Ics4Wrapper,
		in.ChannelKeeper,
		in.PortKeeper,
		in.ScopedKeeper,
		in.MsgRouter,
		authority.String(),
	)
	hostkeeper := hostKeeper.NewKeeper(
		in.Cdc,
		in.Key,
		in.LegacySubspace,
		in.Ics4Wrapper,
		in.ChannelKeeper,
		in.PortKeeper,
		in.AccountKeeper,
		in.ScopedKeeper,
		in.MsgRouter,
		in.QueryRouter,
		authority.String(),
	)
	m := NewAppModule(&controllerkeeper, &hostkeeper)

	return ModuleOutputs{ControllerKeeper: &controllerkeeper, HostKeeper: &hostkeeper, Module: m}
}
