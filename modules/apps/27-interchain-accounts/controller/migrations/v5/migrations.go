package v5

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
)

// MigrateICS27ChannelCapability performs a search on a prefix store using the provided store key and module name.
// It retrieves the associated channel capability index and reassigns ownership to the ICS27 controller submodule.
func MigrateICS27ChannelCapability(
	ctx sdk.Context,
	memStoreKey storetypes.StoreKey,
	capabilityKeeper *capabilitykeeper.Keeper,
	module string,
) error {
	// construct a prefix store using the x/capability reverse lookup key: {module}/rev/{name} -> index
	keyPrefix := capabilitytypes.RevCapabilityKey(module, fmt.Sprintf("%s/%s/%s", host.KeyChannelCapabilityPrefix, host.KeyPortPrefix, icatypes.PortPrefix))
	prefixStore := prefix.NewStore(ctx.KVStore(memStoreKey), keyPrefix)

	iterator := sdk.KVStorePrefixIterator(prefixStore, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// unmarshal the capability index value and retrieve the set of owners
		index := sdk.BigEndianToUint64(iterator.Value())
		owners, found := capabilityKeeper.GetOwners(ctx, index)
		if !found {
			return sdkerrors.Wrapf(capabilitytypes.ErrCapabilityOwnersNotFound, "no owners found for capability at index: %d", index)
		}

		// reconstruct the capability name using the prefixes and iterator key
		name := fmt.Sprintf("%s/%s/%s%s", host.KeyChannelCapabilityPrefix, host.KeyPortPrefix, icatypes.PortPrefix, string(iterator.Key()))
		prevOwner := capabilitytypes.NewOwner(module, name)
		newOwner := capabilitytypes.NewOwner(types.SubModuleName, name)

		// remove the existing module owner
		owners.Remove(prevOwner)
		prefixStore.Delete(iterator.Key())

		// add the controller submodule to the set of owners and initialise the capability
		if err := owners.Set(newOwner); err != nil {
			return err
		}

		capabilityKeeper.SetOwners(ctx, index, owners)
		capabilityKeeper.InitializeCapability(ctx, index, owners)
	}

	return nil
}
