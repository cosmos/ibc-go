package keeper

import (
	"fmt"
	"strings"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	internaltypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/internal/types"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{
		keeper: keeper,
	}
}

// MigrateParams migrates the transfer module's parameters from the x/params to self store.
func (m Migrator) MigrateParams(ctx sdk.Context) error {
	var params types.Params
	m.keeper.legacySubspace.GetParamSet(ctx, &params)

	m.keeper.SetParams(ctx, params)
	m.keeper.Logger(ctx).Info("successfully migrated transfer app self-manage params")
	return nil
}

// MigrateDenomMetadata sets token metadata for all the IBC denom traces
func (m Migrator) MigrateDenomMetadata(ctx sdk.Context) error {
	m.keeper.iterateDenomTraces(ctx,
		func(dt internaltypes.DenomTrace) (stop bool) {
			// check if the metadata for the given denom trace does not already exist
			if !m.keeper.BankKeeper.HasDenomMetaData(ctx, dt.IBCDenom()) {
				m.keeper.setDenomMetadataWithDenomTrace(ctx, dt)
			}
			return false
		})

	m.keeper.Logger(ctx).Info("successfully added metadata to IBC voucher denominations")
	return nil
}

// MigrateTotalEscrowForDenom migrates the total amount of source chain tokens in escrow.
func (m Migrator) MigrateTotalEscrowForDenom(ctx sdk.Context) error {
	var totalEscrowed sdk.Coins
	portID := m.keeper.GetPort(ctx)

	transferChannels := m.keeper.channelKeeper.GetAllChannelsWithPortPrefix(ctx, portID)
	for _, channel := range transferChannels {
		escrowAddress := types.GetEscrowAddress(portID, channel.ChannelId)
		escrowBalances := m.keeper.BankKeeper.GetAllBalances(ctx, escrowAddress)

		totalEscrowed = totalEscrowed.Add(escrowBalances...)
	}

	for _, totalEscrow := range totalEscrowed {
		m.keeper.SetTotalEscrowForDenom(ctx, totalEscrow)
	}

	m.keeper.Logger(ctx).Info("successfully set total escrow", "number of denominations", totalEscrowed.Len())
	return nil
}

// MigrateDenomTraceToDenom migrates storage from using DenomTrace to Denom.
func (m Migrator) MigrateDenomTraceToDenom(ctx sdk.Context) error {
	var (
		denoms      []types.Denom
		denomTraces []internaltypes.DenomTrace
	)
	m.keeper.iterateDenomTraces(ctx,
		func(dt internaltypes.DenomTrace) (stop bool) {
			// convert denomTrace to denom
			denom := types.ExtractDenomFromPath(dt.GetFullDenomPath())
			err := denom.Validate()
			if err != nil {
				panic(err)
			}

			// defense in depth
			if dt.IBCDenom() != denom.IBCDenom() {
				// This migration must not change the SDK coin denom.
				// A panic should occur to prevent the chain from using corrupted state.
				panic(fmt.Errorf("migration will result in corrupted state. expected: %s, got: %s", denom.IBCDenom(), dt.IBCDenom()))
			}

			denoms = append(denoms, denom)
			denomTraces = append(denomTraces, dt)

			return false
		})

	if len(denoms) != len(denomTraces) {
		return fmt.Errorf("length of denoms does not match length of denom traces, %d != %d", len(denoms), len(denomTraces))
	}

	for i := 0; i < len(denoms); i++ {
		m.keeper.SetDenom(ctx, denoms[i])
		m.keeper.deleteDenomTrace(ctx, denomTraces[i])
	}

	return nil
}

// setDenomTrace sets a new {trace hash -> denom trace} pair to the store.
func (k Keeper) setDenomTrace(ctx sdk.Context, denomTrace internaltypes.DenomTrace) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.DenomTraceKey)
	bz := k.cdc.MustMarshal(&denomTrace)

	store.Set(denomTrace.Hash(), bz)
}

// deleteDenomTrace deletes the denom trace
func (k Keeper) deleteDenomTrace(ctx sdk.Context, denomTrace internaltypes.DenomTrace) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.DenomTraceKey)
	store.Delete(denomTrace.Hash())
}

// iterateDenomTraces iterates over the denomination traces in the store
// and performs a callback function.
func (k Keeper) iterateDenomTraces(ctx sdk.Context, cb func(denomTrace internaltypes.DenomTrace) bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, types.DenomTraceKey)

	defer sdk.LogDeferred(k.Logger(ctx), func() error { return iterator.Close() })
	for ; iterator.Valid(); iterator.Next() {
		var denomTrace internaltypes.DenomTrace
		k.cdc.MustUnmarshal(iterator.Value(), &denomTrace)

		if cb(denomTrace) {
			break
		}
	}
}

// setDenomMetadataWithDenomTrace sets an IBC token's denomination metadata
func (k Keeper) setDenomMetadataWithDenomTrace(ctx sdk.Context, denomTrace internaltypes.DenomTrace) {
	metadata := banktypes.Metadata{
		Description: fmt.Sprintf("IBC token from %s", denomTrace.GetFullDenomPath()),
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    denomTrace.BaseDenom,
				Exponent: 0,
			},
		},
		// Setting base as IBC hash denom since bank keepers's SetDenomMetadata uses
		// Base as key path and the IBC hash is what gives this token uniqueness
		// on the executing chain
		Base:    denomTrace.IBCDenom(),
		Display: denomTrace.GetFullDenomPath(),
		Name:    fmt.Sprintf("%s IBC token", denomTrace.GetFullDenomPath()),
		Symbol:  strings.ToUpper(denomTrace.BaseDenom),
	}

	k.BankKeeper.SetDenomMetaData(ctx, metadata)
}
