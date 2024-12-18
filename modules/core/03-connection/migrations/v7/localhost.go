package v7

import "context"

// MigrateLocalhostConnection creates the sentinel localhost connection end to enable
// localhost ibc functionality.
func MigrateLocalhostConnection(ctx context.Context, connectionKeeper ConnectionKeeper) {
	connectionKeeper.CreateSentinelLocalhostConnection(ctx)
}
