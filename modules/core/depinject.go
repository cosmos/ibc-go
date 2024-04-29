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
	channelkeeper "github.com/cosmos/ibc-go/v8/modules/core/04-channel/keeper"
	portkeeper "github.com/cosmos/ibc-go/v8/modules/core/05-port/keeper"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	"github.com/cosmos/ibc-go/v8/modules/core/types"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

var _ depinject.OnePerModuleType = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

func init() {
	appmodule.Register(
		&modulev1.Module{},
		appmodule.Provide(ProvideModule),
		appmodule.Invoke(InvokeAddAppRoutes, InvokeAddClientRoutes),
	)
}

// ModuleInputs defines the core module inputs for depinject.
type ModuleInputs struct {
	depinject.In

	Config *modulev1.Module
	Cdc    codec.Codec
	Key    *storetypes.KVStoreKey

	CapabilityKeeper *capabilitykeeper.Keeper
	StakingKeeper    clienttypes.StakingKeeper
	UpgradeKeeper    clienttypes.UpgradeKeeper

	// LegacySubspace is used solely for migration of x/params managed parameters
	LegacySubspace paramtypes.Subspace `optional:"true"`
}

// ModuleOutputs defines the core module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	Module appmodule.AppModule

	IBCKeeper       *ibckeeper.Keeper
	ScopedIBCKeeper types.ScopedIBCKeeper

	ChannelKeeper *channelkeeper.Keeper
	PortKeeper    *portkeeper.Keeper
}

// ProvideModule defines a depinject provider function to supply the module dependencies and return its outputs.
func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	scopedKeeper := in.CapabilityKeeper.ScopeToModule(exported.ModuleName)

	keeper := ibckeeper.NewKeeper(
		in.Cdc,
		in.Key,
		in.LegacySubspace,
		ibctm.NewConsensusHost(in.StakingKeeper), // NOTE: need to find a way to inject a ConsensusHost into DI container created outside context of app module.
		in.UpgradeKeeper,
		scopedKeeper,
		authority.String(),
	)
	m := NewAppModule(keeper)

	return ModuleOutputs{
		Module:          m,
		IBCKeeper:       keeper,
		ChannelKeeper:   keeper.ChannelKeeper,
		PortKeeper:      keeper.PortKeeper,
		ScopedIBCKeeper: types.ScopedIBCKeeper{ScopedKeeper: scopedKeeper},
	}
}

// InvokeAddAppRoutes defines a depinject Invoker for registering ibc application modules on the core ibc application router.
func InvokeAddAppRoutes(keeper *ibckeeper.Keeper, appRoutes []porttypes.IBCModuleRoute) {
	ibcRouter := porttypes.NewRouter()
	for _, route := range appRoutes {
		ibcRouter.AddRoute(route.Name, route.IBCModule)
	}

	keeper.SetRouter(ibcRouter)
}

// InvokeAddClientRoutes defines a depinject Invoker for registering ibc light client modules on the core ibc client router.
// TODO: Maybe this should align with app router. i.e. create router here, add routes, and set on ibc keeper.
// For app_v1 this would be the same approach, just create clientRouter in app.go instead of implicit creation inside of ibc.NewKeeper()
func InvokeAddClientRoutes(keeper *ibckeeper.Keeper, clientRoutes map[string]clienttypes.LightClientModuleWrapper) {
	router := keeper.ClientKeeper.GetRouter()
	for modName, route := range clientRoutes {
		router.AddRoute(modName, route.LightClientModule)
	}
}
