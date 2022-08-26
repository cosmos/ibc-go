package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
)

// MigrateChannelCapability takes in global capability keeper and auth module scoped keeper name,
// iterates through all capabilities and checks if one of the owners is the auth module,
// if so, replaces the capabilities owner with the controller module scoped keeper
func MigrateChannelCapability(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	memStoreKey storetypes.StoreKey,
	capabilityKeeper capabilitykeeper.Keeper,
	authModule string,
) error {
	keyPrefix := capabilitytypes.RevCapabilityKey(authModule, fmt.Sprintf("%s/%s/%s", host.KeyChannelCapabilityPrefix, host.KeyPortPrefix, icatypes.PortPrefix))
	prefixStore := prefix.NewStore(ctx.KVStore(memStoreKey), keyPrefix)
	iterator := sdk.KVStorePrefixIterator(prefixStore, nil)

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		key := string(iterator.Key())

		name := fmt.Sprintf("%s/%s/%s%s", host.KeyChannelCapabilityPrefix, host.KeyPortPrefix, types.PortPrefix, key)
		capOwner := capabilitytypes.NewOwner(authModule, name)

		ctx.Logger().Info("migrating ibc channel capability", "owner", capOwner.String())

		index := sdk.BigEndianToUint64(iterator.Value())

		capOwners, found := capabilityKeeper.GetOwners(ctx, index)
		if !found {
			panic(fmt.Sprintf("no owners for capability at index: %d", index))
		}

		capOwners.Remove(capOwner)
		capOwners.Set(capabilitytypes.NewOwner(types.ModuleName, name))

		capabilityKeeper.SetOwners(ctx, index, capOwners)
	}

	return nil
}
