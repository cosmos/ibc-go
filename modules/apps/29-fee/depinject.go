package fee

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	modulev1 "github.com/cosmos/ibc-go/api/ibc/applications/fee/module/v1"
	keeper "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/keeper"
	"github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
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

type ModuleInputs struct {
	depinject.In

	Config *modulev1.Module
	Cdc    codec.Codec
	Key    *storetypes.KVStoreKey

	Ics4Wrapper   porttypes.ICS4Wrapper
	ChannelKeeper types.ChannelKeeper
	PortKeeper    types.PortKeeper
	AuthKeeper    types.AccountKeeper
	BankKeeper    types.BankKeeper

	// LegacySubspace is used solely for migration of x/params managed parameters
	LegacySubspace paramtypes.Subspace `optional:"true"`
}

type ModuleOutputs struct {
	depinject.Out

	FeeKeeper *keeper.Keeper
	Module    appmodule.AppModule
}

// ProvideModule returns the  29-fee module outputs for dependency injection
func ProvideModule(in ModuleInputs) ModuleOutputs {
	feeKeeper := keeper.NewKeeper(
		in.Cdc,
		in.Key,
		in.Ics4Wrapper,
		in.ChannelKeeper,
		in.PortKeeper,
		in.AuthKeeper,
		in.BankKeeper,
	)
	m := NewAppModule(feeKeeper)

	return ModuleOutputs{FeeKeeper: &feeKeeper, Module: m}
}
