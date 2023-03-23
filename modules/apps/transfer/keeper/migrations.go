package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// MigrateTraces migrates the DenomTraces to the correct format, accounting for slashes in the BaseDenom.
func (m Migrator) MigrateTraces(ctx sdk.Context) error {
	// list of traces that must replace the old traces in store
	var newTraces []types.DenomTrace
	m.keeper.IterateDenomTraces(ctx,
		func(dt types.DenomTrace) (stop bool) {
			// check if the new way of splitting FullDenom
			// is the same as the current DenomTrace.
			// If it isn't then store the new DenomTrace in the list of new traces.
			newTrace := types.ParseDenomTrace(dt.GetFullDenomPath())
			err := newTrace.Validate()
			if err != nil {
				panic(err)
			}

			if dt.IBCDenom() != newTrace.IBCDenom() {
				// The new form of parsing will result in a token denomination change.
				// A bank migration is required. A panic should occur to prevent the
				// chain from using corrupted state.
				panic(fmt.Sprintf("migration will result in corrupted state. Previous IBC token (%s) requires a bank migration. Expected denom trace (%s)", dt, newTrace))
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
	return nil
}

// MigrateTotalEscrowForDenom migrates the total amount of source chain tokens in escrow.
func (m Migrator) MigrateTotalEscrowForDenom(ctx sdk.Context) error {
	tokenDenomToTotalEscrowMap := make(map[string]math.Int)

	transferChannels := m.keeper.channelKeeper.GetAllChannelsWithPortPrefix(ctx, types.PortID)
	for _, channel := range transferChannels {
		escrowAddress := types.GetEscrowAddress(types.PortID, channel.ChannelId)
		escrowBalances := m.keeper.bankKeeper.GetAllBalances(ctx, escrowAddress)

		for _, escrowBalance := range escrowBalances {
			if val, ok := tokenDenomToTotalEscrowMap[escrowBalance.Denom]; ok {
				tokenDenomToTotalEscrowMap[escrowBalance.Denom] = val.Add(escrowBalance.Amount)
			} else {
				tokenDenomToTotalEscrowMap[escrowBalance.Denom] = escrowBalance.Amount
			}
		}
	}

	if len(tokenDenomToTotalEscrowMap) != 0 {
		for denom, amount := range tokenDenomToTotalEscrowMap {
			m.keeper.SetTotalEscrowForDenom(ctx, denom, amount)
		}
	}

	return nil
}

func equalTraces(dtA, dtB types.DenomTrace) bool {
	return dtA.BaseDenom == dtB.BaseDenom && dtA.Path == dtB.Path
}
