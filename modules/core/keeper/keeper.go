package keeper

import (
	"errors"
	"reflect"
	"strings"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/codec"

	clientkeeper "github.com/cosmos/ibc-go/v9/modules/core/02-client/keeper"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectionkeeper "github.com/cosmos/ibc-go/v9/modules/core/03-connection/keeper"
	channelkeeper "github.com/cosmos/ibc-go/v9/modules/core/04-channel/keeper"
	portkeeper "github.com/cosmos/ibc-go/v9/modules/core/05-port/keeper"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v9/modules/core/types"
)

// Keeper defines each ICS keeper for IBC
type Keeper struct {
	appmodule.Environment

	ClientKeeper     *clientkeeper.Keeper
	ConnectionKeeper *connectionkeeper.Keeper
	ChannelKeeper    *channelkeeper.Keeper
	PortKeeper       *portkeeper.Keeper

	cdc codec.BinaryCodec

	authority string
}

// NewKeeper creates a new ibc Keeper
func NewKeeper(
	cdc codec.BinaryCodec, env appmodule.Environment, paramSpace types.ParamSubspace,
	upgradeKeeper clienttypes.UpgradeKeeper, authority string,
) *Keeper {
	// panic if any of the keepers passed in is empty
	if isEmpty(upgradeKeeper) {
		panic(errors.New("cannot initialize IBC keeper: empty upgrade keeper"))
	}

	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	clientKeeper := clientkeeper.NewKeeper(cdc, env, paramSpace, upgradeKeeper)
	connectionKeeper := connectionkeeper.NewKeeper(cdc, env, paramSpace, clientKeeper)
	portKeeper := portkeeper.NewKeeper()
	channelKeeper := channelkeeper.NewKeeper(cdc, env, clientKeeper, connectionKeeper)

	return &Keeper{
		Environment:      env,
		cdc:              cdc,
		ClientKeeper:     clientKeeper,
		ConnectionKeeper: connectionKeeper,
		ChannelKeeper:    channelKeeper,
		PortKeeper:       portKeeper,
		authority:        authority,
	}
}

// Codec returns the IBC module codec.
func (k *Keeper) Codec() codec.BinaryCodec {
	return k.cdc
}

// SetRouter sets the Router in IBC Keeper and seals it. The method panics if
// there is an existing router that's already sealed.
func (k *Keeper) SetRouter(rtr *porttypes.Router) {
	if k.PortKeeper.Router != nil && k.PortKeeper.Router.Sealed() {
		panic(errors.New("cannot reset a sealed router"))
	}

	k.PortKeeper.Router = rtr
	k.PortKeeper.Router.Seal()
}

// GetAuthority returns the ibc module's authority.
func (k *Keeper) GetAuthority() string {
	return k.authority
}

// isEmpty checks if the interface is an empty struct or a pointer pointing
// to an empty struct
func isEmpty(keeper interface{}) bool {
	switch reflect.TypeOf(keeper).Kind() {
	case reflect.Ptr:
		if reflect.ValueOf(keeper).Elem().IsZero() {
			return true
		}
	default:
		if reflect.ValueOf(keeper).IsZero() {
			return true
		}
	}
	return false
}
