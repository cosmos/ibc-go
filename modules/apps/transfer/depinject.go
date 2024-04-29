package transfer

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	modulev1 "github.com/cosmos/ibc-go/api/ibc/applications/transfer/module/v1"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
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

// ModuleInputs defines the transfer module inputs for depinject.
type ModuleInputs struct {
	depinject.In

	Config *modulev1.Module
	Cdc    codec.Codec
	Key    *storetypes.KVStoreKey

	ChannelKeeper types.ChannelKeeper
	Ics4Wrapper   porttypes.ICS4Wrapper
	PortKeeper    types.PortKeeper

	AuthKeeper       types.AccountKeeper
	BankKeeper       types.BankKeeper
	CapabilityKeeper *capabilitykeeper.Keeper

	// LegacySubspace is used solely for migration of x/params managed parameters
	LegacySubspace paramtypes.Subspace `optional:"true"`
}

// ModuleOutputs defines the transfer module outputs for depinject.
type ModuleOutputs struct {
	depinject.Out

	ScopedKeeper   types.ScopedTransferKeeper
	TransferKeeper keeper.Keeper
	Module         appmodule.AppModule
	IBCModuleRoute porttypes.IBCModuleRoute
}

// ProvideModule returns the transfer module outputs for dependency injection
func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	scopedKeeper := in.CapabilityKeeper.ScopeToModule(types.ModuleName)
	transferKeeper := keeper.NewKeeper(
		in.Cdc,
		in.Key,
		in.LegacySubspace,
		in.Ics4Wrapper,
		in.ChannelKeeper,
		in.PortKeeper,
		in.AuthKeeper,
		in.BankKeeper,
		scopedKeeper,
		authority.String(),
	)

	m := NewAppModule(transferKeeper)
	ibcModule := NewIBCModule(transferKeeper)

	return ModuleOutputs{
		ScopedKeeper:   types.ScopedTransferKeeper{ScopedKeeper: scopedKeeper},
		TransferKeeper: transferKeeper,
		Module:         m,
		IBCModuleRoute: porttypes.IBCModuleRoute{Name: types.ModuleName, IBCModule: ibcModule},
	}
}
