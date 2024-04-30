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
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller"
	controllerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	controllertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host"
	hostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	hosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
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

	// TODO: Config should define `controller_enabled` and `host_enabled` vars for keeper/module setup
	Config *modulev1.Module
	Cdc    codec.Codec

	// TODO: runtime seems to expect that a module contains a single kvstore key.
	ControllerKey *storetypes.KVStoreKey
	HostKey       *storetypes.KVStoreKey

	Ics4Wrapper      porttypes.ICS4Wrapper
	ChannelKeeper    types.ChannelKeeper
	PortKeeper       types.PortKeeper
	CapabilityKeeper *capabilitykeeper.Keeper
	AccountKeeper    types.AccountKeeper

	MsgRouter types.MessageRouter
	// TODO(remove optional): GRCPQueryRouter is not outputted into DI container on v0.50. It is on main.
	QueryRouter types.QueryRouter `optional:"true"`

	// LegacySubspace is used solely for migration of x/params managed parameters
	LegacySubspace paramtypes.Subspace `optional:"true"`
}

// ModuleOutputs defines the 27-interchain-accounts module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	ControllerKeeper *controllerkeeper.Keeper
	HostKeeper       *hostkeeper.Keeper
	Module           appmodule.AppModule
	IBCModuleRoutes  []porttypes.IBCModuleRoute
}

// ProvideModule returns the 27-interchain-accounts outputs for dependency injection
func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	scopedControllerKeeper := in.CapabilityKeeper.ScopeToModule(controllertypes.SubModuleName)
	controllerKeeper := controllerkeeper.NewKeeper(
		in.Cdc,
		in.ControllerKey,
		in.LegacySubspace,
		in.Ics4Wrapper,
		in.ChannelKeeper,
		in.PortKeeper,
		scopedControllerKeeper,
		in.MsgRouter,
		authority.String(),
	)

	scopedHostKeeper := in.CapabilityKeeper.ScopeToModule(hosttypes.SubModuleName)
	hostKeeper := hostkeeper.NewKeeper(
		in.Cdc,
		in.HostKey,
		in.LegacySubspace,
		in.Ics4Wrapper,
		in.ChannelKeeper,
		in.PortKeeper,
		in.AccountKeeper,
		scopedHostKeeper,
		in.MsgRouter,
		in.QueryRouter,
		authority.String(),
	)
	m := NewAppModule(&controllerKeeper, &hostKeeper)

	controllerModule := controller.NewIBCMiddleware(nil, controllerKeeper)
	hostModule := host.NewIBCModule(hostKeeper)

	return ModuleOutputs{
		ControllerKeeper: &controllerKeeper,
		HostKeeper:       &hostKeeper,
		Module:           m,
		IBCModuleRoutes: []porttypes.IBCModuleRoute{
			{
				Name: controllertypes.SubModuleName, IBCModule: controllerModule,
			},
			{
				Name: hosttypes.SubModuleName, IBCModule: hostModule,
			},
		},
	}
}
