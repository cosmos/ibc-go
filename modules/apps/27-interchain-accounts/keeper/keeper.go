package keeper

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

// Keeper defines the IBC interchain accounts keeper
type Keeper struct {
	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	portKeeper types.PortKeeper
}

// NewKeeper creates a new interchain accounts Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey,
	portKeeper types.PortKeeper,
) Keeper {
	return Keeper{
		storeKey:   key,
		cdc:        cdc,
		portKeeper: portKeeper,
	}
}

// Logger returns the application logger, scoped to the associated module
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s-%s", host.ModuleName, types.ModuleName))
}

// prefixStore returns isolated prefix store for each ICA submodule so they can read/write in separate
// namespace without being able to read/write to each other's data.
func (k Keeper) prefixStore(ctx sdk.Context, storePrefix string) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), []byte(storePrefix))
}

// GetAllPorts returns all ports to which a interchain accounts submodule is bound. Used in ExportGenesis
func (k Keeper) GetAllPorts(ctx sdk.Context, storePrefix string) []string {
	store := k.prefixStore(ctx, storePrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.PortKeyPrefix))
	defer iterator.Close()

	var ports []string
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		ports = append(ports, keySplit[1])
	}

	return ports
}

// BindPort stores the provided portID and binds to it, returning the associated capability. The
// binded port is stored in state under the associated interchain accounts submodule.
func (k Keeper) BindPort(ctx sdk.Context, storePrefix, portID string) *capabilitytypes.Capability {
	store := k.prefixStore(ctx, storePrefix)
	store.Set(types.KeyPort(portID), []byte{0x01})

	return k.portKeeper.BindPort(ctx, portID)
}

// GetActiveChannelID retrieves the active channelID from the store keyed by the provided portID
// for a given interchain accounts submodule.
func (k Keeper) GetActiveChannelID(ctx sdk.Context, storePrefix, portID string) (string, bool) {
	store := k.prefixStore(ctx, storePrefix)
	key := types.KeyActiveChannel(portID)

	if !store.Has(key) {
		return "", false
	}

	return string(store.Get(key)), true
}

// GetAllActiveChannels returns a list of all active interchain accounts channels and their associated port identifiers
// for a give interchain accounts submodule.
func (k Keeper) GetAllActiveChannels(ctx sdk.Context, storePrefix string) []types.ActiveChannel {
	store := k.prefixStore(ctx, storePrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.ActiveChannelKeyPrefix))
	defer iterator.Close()

	var activeChannels []types.ActiveChannel
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		ch := types.ActiveChannel{
			PortId:    keySplit[1],
			ChannelId: string(iterator.Value()),
		}

		activeChannels = append(activeChannels, ch)
	}

	return activeChannels
}

// SetActiveChannelID stores the active channelID, keyed by the provided portID for a given
// interchain accounts submodule.
func (k Keeper) SetActiveChannelID(ctx sdk.Context, storePrefix, portID, channelID string) {
	store := k.prefixStore(ctx, storePrefix)
	store.Set(types.KeyActiveChannel(portID), []byte(channelID))
}

// DeleteActiveChannelID removes the active channel keyed by the provided portID stored in state
// for a given interchain accounts submodule.
func (k Keeper) DeleteActiveChannelID(ctx sdk.Context, storePrefix, portID string) {
	store := k.prefixStore(ctx, storePrefix)
	store.Delete(types.KeyActiveChannel(portID))
}

// IsActiveChannel returns true if there exists an active channel (for a given interchain accounts submodule)
// for the provided portID, otherwise false
func (k Keeper) IsActiveChannel(ctx sdk.Context, storePrefix, portID string) bool {
	_, ok := k.GetActiveChannelID(ctx, storePrefix, portID)
	return ok
}

// GetInterchainAccountAddress retrieves the InterchainAccount address from the store keyed by the provided portID
// for a given interchain accounts submodule.
func (k Keeper) GetInterchainAccountAddress(ctx sdk.Context, storePrefix, portID string) (string, bool) {
	store := k.prefixStore(ctx, storePrefix)
	key := types.KeyOwnerAccount(portID)

	if !store.Has(key) {
		return "", false
	}

	return string(store.Get(key)), true
}

// GetAllInterchainAccounts returns a list of all registered interchain account addresses and their associated controller port identifiers
// for a given interchain accounts submodule.
func (k Keeper) GetAllInterchainAccounts(ctx sdk.Context, storePrefix string) []types.RegisteredInterchainAccount {
	store := k.prefixStore(ctx, storePrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.OwnerKeyPrefix))

	var interchainAccounts []types.RegisteredInterchainAccount
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		acc := types.RegisteredInterchainAccount{
			PortId:         keySplit[1],
			AccountAddress: string(iterator.Value()),
		}

		interchainAccounts = append(interchainAccounts, acc)
	}

	return interchainAccounts
}

// SetInterchainAccountAddress stores the InterchainAccount address, keyed by the associated portID
// for a given interchain accounts submodule.
func (k Keeper) SetInterchainAccountAddress(ctx sdk.Context, storePrefix, portID string, address string) {
	store := k.prefixStore(ctx, storePrefix)
	store.Set(types.KeyOwnerAccount(portID), []byte(address))
}
