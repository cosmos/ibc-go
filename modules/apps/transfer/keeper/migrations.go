package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates from version 1 to 2.
// This migration will update the denom
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	var iterErr error

	// list of traces that must replace the old traces in store
	var newTraces []types.DenomTrace
	m.keeper.IterateDenomTraces(ctx,
		func(dt types.DenomTrace) (stop bool) {
			// check if the new way of splitting FullDenom
			// into Trace and BaseDenom passes validation and
			// is the same as the current DenomTrace.
			// If it isn't then store the new DenomTrace in the list of new traces.
			newTrace := types.ParseDenomTrace(dt.GetFullDenomPath())
			err := newTrace.Validate()
			if err != nil {
				iterErr = err
				return true
			}
			if !equalTraces(newTrace, dt) {
				newTraces = append(newTraces, newTrace)
			}
			return false
		})

	// replace the outdated traces with the new trace information
	for _, nt := range newTraces {
		m.keeper.SetDenomTrace(ctx, nt)
	}
	return iterErr
}

func equalTraces(dtA, dtB types.DenomTrace) bool {
	return dtA.BaseDenom == dtB.BaseDenom && dtA.Path == dtB.Path
}
