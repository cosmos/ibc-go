package v7

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v6/modules/core/03-connection/types"
)

// MigrateLocalhostConnectionEnd creates the sentinel localhost connection end to enable
// localhost ibc functionality.
func MigrateLocalhostConnectionEnd(ctx sdk.Context, connectionKeeper ConnectionKeeper) {
	localhostConnection := connectionKeeper.CreateSentinelLocalhostConnection()
	connectionKeeper.SetConnection(ctx, types.LocalhostID, localhostConnection)
}
