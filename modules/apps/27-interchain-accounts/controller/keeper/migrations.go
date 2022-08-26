package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
)

// MigrateChannelCapability performs a search on a prefix store using the provided authModule name,
// retrieves the associated capability index and reassigns ownership to the ICS27 controller submodule
func MigrateChannelCapability(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	memStoreKey storetypes.StoreKey,
	capabilityKeeper capabilitykeeper.Keeper,
	authModule string,
) error {
	keyPrefix := capabilitytypes.RevCapabilityKey(authModule, fmt.Sprintf("%s/%s/%s", host.KeyChannelCapabilityPrefix, host.KeyPortPrefix, icatypes.PortPrefix))
	prefixStore := prefix.NewStore(ctx.KVStore(memStoreKey), keyPrefix)
	iterator := sdk.KVStorePrefixIterator(prefixStore, nil)

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		key := string(iterator.Key()) // search prefix is omitted

		// reconstruct the capability name using the prefix and iterator key
		name := fmt.Sprintf("%s/%s/%s%s", host.KeyChannelCapabilityPrefix, host.KeyPortPrefix, icatypes.PortPrefix, key)
		capOwner := capabilitytypes.NewOwner(authModule, name)

		ctx.Logger().Info("migrating ibc channel capability", "owner", capOwner.String())

		index := sdk.BigEndianToUint64(iterator.Value())
		capOwners, found := capabilityKeeper.GetOwners(ctx, index)
		if !found {
			panic(fmt.Sprintf("no owners for capability at index: %d", index))
		}

		// remove the existing auth module owner
		capOwners.Remove(capOwner)

		// add the controller submodule as a new capability owner
		capOwners.Set(capabilitytypes.NewOwner(types.SubModuleName, name))

		capabilityKeeper.SetOwners(ctx, index, capOwners)
	}

	return nil
}
