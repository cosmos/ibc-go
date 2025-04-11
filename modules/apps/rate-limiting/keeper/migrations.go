package keeper

// import (
// 	sdk "github.com/cosmos/cosmos-sdk/types"

// 	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
// )

// // Migrator is a struct for handling in-place state migrations.
// type Migrator struct {
// 	keeper Keeper
// }

// // NewMigrator creates a new Migrator instance.
// func NewMigrator(keeper Keeper) Migrator {
// 	return Migrator{keeper: keeper}
// }

// // MigrateParams migrates the parameters from a legacy param subspace to the proper
// // params module. This function is only required on an upgrade from v1 to v2.
// func (m Migrator) MigrateParams(ctx sdk.Context) error {
// 	// Get the params from the subspace
// 	var params types.Params
// 	m.keeper.paramSpace.GetParamSet(ctx, &params)

// 	// Set the params directly in the keeper
// 	m.keeper.SetParams(ctx, params)

// 	return nil
// }
