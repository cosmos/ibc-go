package keeper

import sdk "github.com/cosmos/cosmos-sdk/types"

// MigrateContractCode is a wrapper around k.migrateContractCode to allow the method to be directly called in tests.
func (k *Keeper) MigrateContractCode(ctx sdk.Context, clientID string, newChecksum, migrateMsg []byte) error {
	return k.migrateContractCode(ctx, clientID, newChecksum, migrateMsg)
}

// GetQueryPlugins is a wrapper around k.getQueryPlugins to allow the method to be directly called in tests.
func (k *Keeper) GetQueryPlugins() QueryPlugins {
	return k.getQueryPlugins()
}

// SetQueryPlugins is a wrapper around k.setQueryPlugins to allow the method to be directly called in tests.
func (k *Keeper) SetQueryPlugins(plugins QueryPlugins) {
	k.setQueryPlugins(plugins)
}
