package upgrades

import (
	"context"

	"cosmossdk.io/core/appmodule"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"
)

const (
	// V8 defines the upgrade name for the ibc-go/v9 upgrade handler.
	V8 = "v8"
	// V8_1 defines the upgrade name for the ibc-go/v9.1 upgrade handler.
	V8_1 = "v8.1"
	// V9 defines the upgrade name for the ibc-go/v9 upgrade handler.
	V9 = "v9"
	// V1010 upgrade handler.
	V10 = "v10"
)

// CreateDefaultUpgradeHandler creates an upgrade handler which can be used for regular upgrade tests
// that do not require special logic
func CreateDefaultUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm appmodule.VersionMap) (appmodule.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
