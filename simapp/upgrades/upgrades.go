package upgrades

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const (
	// V8 defines the upgrade name for the ibc-go/v8 upgrade handler.
	V8 = "v8"
	// V8_1 defines the upgrade name for the ibc-go/v8.1 upgrade handler.
	V8_1 = "v8.1"
	// V10 defines the upgrade name for the ibc-go/v10 upgrade handler.
	V10 = "v10"
	// V11 defines the upgrade name for the ibc-go/v11 upgrade handler.
	V11 = "v11"
	// V11_1 defines the upgrade name for the ibc-go/v11.1 upgrade handler.
	V11_1 = "v11.1"
	// V11_1LegacyPFM defines the upgrade name for direct legacy v10 PFM upgrades to ibc-go/v11.1.
	V11_1LegacyPFM = "v11.1-legacy-pfm"
)

// CreateDefaultUpgradeHandler creates an upgrade handler which can be used for regular upgrade tests
// that do not require special logic
func CreateDefaultUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
