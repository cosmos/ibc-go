package v7

import sdk "github.com/cosmos/cosmos-sdk/types"

// MigrateLocalhostConnectionEnd creates the sentinel localhost connection end to enable
// localhost ibc functionality.
func MigrateLocalhostConnectionEnd(ctx sdk.Context, connectionKeeper ConnectionKeeper) {
	connectionKeeper.CreateLocalhostConnectionEnd(ctx)
}
