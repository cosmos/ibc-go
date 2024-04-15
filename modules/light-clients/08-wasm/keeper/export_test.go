package keeper

import sdk "github.com/cosmos/cosmos-sdk/types"

// MigrateContractCode is a wrapper around k.migrateContractCode to allow the method to be directly called in tests.
func (k Keeper) MigrateContractCode(ctx sdk.Context, clientID string, newChecksum, migrateMsg []byte) error {
	return k.migrateContractCode(ctx, clientID, newChecksum, migrateMsg)
}
