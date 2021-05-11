package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	clientkeeper "github.com/cosmos/ibc-go/modules/core/02-client/keeper"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectionkeeper "github.com/cosmos/ibc-go/modules/core/03-connection/keeper"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channelkeeper "github.com/cosmos/ibc-go/modules/core/04-channel/keeper"
	portkeeper "github.com/cosmos/ibc-go/modules/core/05-port/keeper"
	porttypes "github.com/cosmos/ibc-go/modules/core/05-port/types"
	wasmkeeper "github.com/cosmos/ibc-go/modules/core/28-wasm/keeper"
	"github.com/cosmos/ibc-go/modules/core/types"
)

var _ types.QueryServer = (*Keeper)(nil)

// Keeper defines each ICS keeper for IBC
type Keeper struct {
	// implements gRPC QueryServer interface
	types.QueryServer

	cdc codec.BinaryCodec

	ClientKeeper     clientkeeper.Keeper
	ConnectionKeeper connectionkeeper.Keeper
	ChannelKeeper    channelkeeper.Keeper
	PortKeeper       portkeeper.Keeper
	WasmKeeper       wasmkeeper.Keeper
	Router           *porttypes.Router
}

// NewKeeper creates a new ibc Keeper
func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey, paramSpace paramtypes.Subspace,
	stakingKeeper clienttypes.StakingKeeper, upgradeKeeper clienttypes.UpgradeKeeper,
	scopedKeeper capabilitykeeper.ScopedKeeper,
) *Keeper {
	// register paramSpace at top level keeper
	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		keyTable := clienttypes.ParamKeyTable()
		keyTable.RegisterParamSet(&connectiontypes.Params{})
		paramSpace = paramSpace.WithKeyTable(keyTable)
	}

	clientKeeper := clientkeeper.NewKeeper(cdc, key, paramSpace, stakingKeeper, upgradeKeeper)
	connectionKeeper := connectionkeeper.NewKeeper(cdc, key, paramSpace, clientKeeper)
	portKeeper := portkeeper.NewKeeper(scopedKeeper)
	channelKeeper := channelkeeper.NewKeeper(cdc, key, clientKeeper, connectionKeeper, portKeeper, scopedKeeper)
	wasmKeeper := wasmkeeper.NewKeeper(cdc, key, &wasmkeeper.WASMValidationConfig{
		MaxSizeAllowed: 1024 * 1024,
	})

	return &Keeper{
		cdc:              cdc,
		ClientKeeper:     clientKeeper,
		ConnectionKeeper: connectionKeeper,
		ChannelKeeper:    channelKeeper,
		PortKeeper:       portKeeper,
		WasmKeeper:       wasmKeeper,
	}
}

// Codec returns the IBC module codec.
func (k Keeper) Codec() codec.BinaryCodec {
	return k.cdc
}

// SetRouter sets the Router in IBC Keeper and seals it. The method panics if
// there is an existing router that's already sealed.
func (k *Keeper) SetRouter(rtr *porttypes.Router) {
	if k.Router != nil && k.Router.Sealed() {
		panic("cannot reset a sealed router")
	}
	k.Router = rtr
	k.Router.Seal()
}
