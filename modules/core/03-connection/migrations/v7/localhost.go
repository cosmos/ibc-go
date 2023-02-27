package v7

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// MigrateLocalhostConnection creates the sentinel localhost connection end to enable
// localhost ibc functionality.
func MigrateLocalhostConnection(ctx sdk.Context, connectionKeeper ConnectionKeeper) {
	localhostConnection := connectionKeeper.CreateSentinelLocalhostConnection()
	connectionKeeper.SetConnection(ctx, exported.LocalhostConnectionID, localhostConnection)
}
