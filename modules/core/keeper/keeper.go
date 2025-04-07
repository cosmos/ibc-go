package keeper

import (
	"errors"
	"reflect"
	"strings"

	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"

	clientkeeper "github.com/cosmos/ibc-go/v10/modules/core/02-client/keeper"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2keeper "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/keeper"
	connectionkeeper "github.com/cosmos/ibc-go/v10/modules/core/03-connection/keeper"
	channelkeeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"
	channelkeeperv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/keeper"
	portkeeper "github.com/cosmos/ibc-go/v10/modules/core/05-port/keeper"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v10/modules/core/api"
	"github.com/cosmos/ibc-go/v10/modules/core/types"
)

// Keeper defines each ICS keeper for IBC
type Keeper struct {
	ClientKeeper     *clientkeeper.Keeper
	ClientV2Keeper   *clientv2keeper.Keeper
	ConnectionKeeper *connectionkeeper.Keeper
	ChannelKeeper    *channelkeeper.Keeper
	ChannelKeeperV2  *channelkeeperv2.Keeper
	PortKeeper       *portkeeper.Keeper

	cdc codec.BinaryCodec

	authority string
}

// NewKeeper creates a new ibc Keeper
func NewKeeper(
	cdc codec.BinaryCodec, storeService corestore.KVStoreService, paramSpace types.ParamSubspace,
	upgradeKeeper clienttypes.UpgradeKeeper, authority string,
) *Keeper {
	// panic if any of the keepers passed in is empty
	if isEmpty(upgradeKeeper) {
		panic(errors.New("cannot initialize IBC keeper: empty upgrade keeper"))
	}

	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	clientKeeper := clientkeeper.NewKeeper(cdc, storeService, paramSpace, upgradeKeeper)
	clientV2Keeper := clientv2keeper.NewKeeper(cdc, clientKeeper)
	connectionKeeper := connectionkeeper.NewKeeper(cdc, storeService, paramSpace, clientKeeper)
	portKeeper := portkeeper.NewKeeper()
	channelKeeper := channelkeeper.NewKeeper(cdc, storeService, clientKeeper, connectionKeeper)
	channelKeeperV2 := channelkeeperv2.NewKeeper(cdc, storeService, clientKeeper, clientV2Keeper, channelKeeper, connectionKeeper)

	return &Keeper{
		cdc:              cdc,
		ClientKeeper:     clientKeeper,
		ClientV2Keeper:   clientV2Keeper,
		ConnectionKeeper: connectionKeeper,
		ChannelKeeper:    channelKeeper,
		ChannelKeeperV2:  channelKeeperV2,
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

// SetRouterV2 sets the v2 router for the IBC Keeper.
func (k *Keeper) SetRouterV2(rtr *api.Router) {
	k.ChannelKeeperV2.Router = rtr
}

// GetAuthority returns the ibc module's authority.
func (k *Keeper) GetAuthority() string {
	return k.authority
}

// isEmpty checks if the interface is an empty struct or a pointer pointing
// to an empty struct
func isEmpty(keeper any) bool {
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
