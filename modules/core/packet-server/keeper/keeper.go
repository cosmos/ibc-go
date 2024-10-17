package keeper

import (
	"context"
	"fmt"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

// Keeper defines the packet keeper. It wraps the client and channel keepers.
// It does not manage its own store.
type Keeper struct {
	cdc           codec.BinaryCodec
	storeService  corestore.KVStoreService
	ChannelKeeper types.ChannelKeeper
	ClientKeeper  types.ClientKeeper
}

// NewKeeper creates a new packet keeper
func NewKeeper(cdc codec.BinaryCodec, storeService corestore.KVStoreService, channelKeeper types.ChannelKeeper, clientKeeper types.ClientKeeper) *Keeper {
	return &Keeper{
		cdc:           cdc,
		storeService:  storeService,
		ChannelKeeper: channelKeeper,
		ClientKeeper:  clientKeeper,
	}
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: https://github.com/cosmos/ibc-go/issues/5917
	return sdkCtx.Logger().With("module", "x/"+exported.ModuleName+"/"+types.SubModuleName)
}

func (k Keeper) ChannelStore(ctx context.Context, channelID string) storetypes.KVStore {
	channelPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyChannelStorePrefix, channelID))
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), channelPrefix)
}

// SetChannel sets the Channel for a given client identifier.
func (k *Keeper) SetChannel(ctx context.Context, clientID string, channel types.Channel) {
	bz := k.cdc.MustMarshal(&channel)
	k.ChannelStore(ctx, clientID).Set([]byte(types.ChannelKey), bz)
}

// GetChannel gets the Channel for a given client identifier.
func (k *Keeper) GetChannel(ctx context.Context, clientID string) (types.Channel, bool) {
	store := k.ChannelStore(ctx, clientID)
	bz := store.Get([]byte(types.ChannelKey))
	if len(bz) == 0 {
		return types.Channel{}, false
	}

	var channel types.Channel
	k.cdc.MustUnmarshal(bz, &channel)
	return channel, true
}
