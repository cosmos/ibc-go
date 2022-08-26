package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
)

// MigrateChannelCapability takes in global capability keeper and auth module scoped keeper name,
// iterates through all capabilities and checks if one of the owners is the auth module,
// if so, replaces the capabilities owner with the controller module scoped keeper
func MigrateChannelCapability(ctx sdk.Context, capabilityKeeper capabilitykeeper.Keeper, authModule string) error {

	// iterate controller port prefix to get name to construct owner

	latestIndex := capabilityKeeper.GetLatestIndex(ctx)

	for i := 1; i <= int(latestIndex); i++ {
		capOwners, found := capabilityKeeper.GetOwners(ctx, uint64(i))
		if !found {
			continue
		}

		// index, found := capOwners.Get(owner)

		// for _, owner := range capOwners.GetOwners() {
		// 	if owner.Module == authModule {
		// 		newOwner := capabilitytypes.NewOwner(types.SubModuleName, "todo")

		// 		capOwners := capabilitytypes.NewCapabilityOwners()

		// 		capabilityKeeper.SetOwners(ctx, uint64(i), *capOwners)
		// 	}
		// }
	}

	return nil
}
