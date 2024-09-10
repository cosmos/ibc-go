package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

// Keeper defines the packet keeper. It wraps the client and channel keepers.
// It does not manage its own store.
type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      storetypes.StoreKey
	ChannelKeeper types.ChannelKeeper
	ClientKeeper  types.ClientKeeper
}

// NewKeeper creates a new packet keeper
func NewKeeper(cdc codec.BinaryCodec, storeKey storetypes.StoreKey, channelKeeper types.ChannelKeeper, clientKeeper types.ClientKeeper) *Keeper {
	return &Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		ChannelKeeper: channelKeeper,
		ClientKeeper:  clientKeeper,
	}
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+exported.ModuleName+"/"+types.SubModuleName)
}

func (k Keeper) ChannelStore(ctx sdk.Context, channelID string) storetypes.KVStore {
	channelPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyChannelStorePrefix, channelID))
	return prefix.NewStore(ctx.KVStore(k.storeKey), channelPrefix)
}

// SetCounterparty sets the Counterparty for a given client identifier.
func (k *Keeper) SetCounterparty(ctx sdk.Context, clientID string, counterparty types.Counterparty) {
	bz := k.cdc.MustMarshal(&counterparty)
	k.ChannelStore(ctx, clientID).Set([]byte(types.CounterpartyKey), bz)
}

// GetCounterparty gets the Counterparty for a given client identifier.
func (k *Keeper) GetCounterparty(ctx sdk.Context, clientID string) (types.Counterparty, bool) {
	store := k.ChannelStore(ctx, clientID)
	bz := store.Get([]byte(types.CounterpartyKey))
	if len(bz) == 0 {
		return types.Counterparty{}, false
	}

	var counterparty types.Counterparty
	k.cdc.MustUnmarshal(bz, &counterparty)
	return counterparty, true
}
