package keeper

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	internaltypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/internal/types"
	"github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	"github.com/cosmos/ibc-go/v11/modules/core/exported"
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

// MigrateDenomMetadata sets token metadata for all the IBC denom traces
func (m Migrator) MigrateDenomMetadata(ctx sdk.Context) error {
	m.keeper.iterateDenomTraces(ctx,
		func(dt internaltypes.DenomTrace) bool {
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

// MigrateChannelEscrow initializes per-channel and per-client escrow accounting from bank balances.
func (m Migrator) MigrateChannelEscrow(ctx sdk.Context) error {
	portID := m.keeper.GetPort(ctx)
	// Use a map to deduplicate identifiers that may be discovered through more than one source
	// before sorting them for deterministic iteration.
	identifiers := map[string]struct{}{exported.LocalhostClientID: {}}
	// IBC v2 aliases are keyed only by v1 channel ID and may originate from any port.
	// An empty prefix returns all v1 channels so every possible aliased escrow identifier is included.
	for _, channel := range m.keeper.channelKeeper.GetAllChannelsWithPortPrefix(ctx, "") {
		identifiers[channel.ChannelId] = struct{}{}
	}
	for _, client := range m.keeper.clientKeeper.GetAllGenesisClients(ctx) {
		identifiers[client.ClientId] = struct{}{}
	}

	// Sort the identifiers to ensure deterministic iteration order.
	sortedIdentifiers := make([]string, 0, len(identifiers))
	for identifier := range identifiers {
		sortedIdentifiers = append(sortedIdentifiers, identifier)
	}
	slices.Sort(sortedIdentifiers)

	channelEscrows := make([]types.ChannelEscrow, 0, len(sortedIdentifiers))
	var calculatedTotal sdk.Coins
	for _, identifier := range sortedIdentifiers {
		escrowAddress := types.GetEscrowAddress(portID, identifier)
		balances := m.keeper.BankKeeper.GetAllBalances(ctx, escrowAddress)
		if !balances.Empty() {
			channelEscrows = append(channelEscrows, types.ChannelEscrow{ChannelOrClientId: identifier, Tokens: balances})
		}
		calculatedTotal = calculatedTotal.Add(balances...)
	}

	existingTotal := m.keeper.GetAllTotalEscrowed(ctx)
	if !calculatedTotal.Equal(existingTotal) {
		return fmt.Errorf("calculated channel escrow %s does not match existing total escrow %s", calculatedTotal, existingTotal)
	}

	if err := m.keeper.ChannelEscrows.Clear(ctx, nil); err != nil {
		return err
	}
	for _, escrow := range channelEscrows {
		for _, coin := range escrow.Tokens {
			m.keeper.SetChannelEscrowForDenom(ctx, escrow.ChannelOrClientId, coin)
		}
	}

	m.keeper.Logger(ctx).Info("successfully set per-channel escrow", "number of channels or clients", len(channelEscrows), "number of denominations", calculatedTotal.Len())
	return nil
}

// MigrateDenomTraceToDenom migrates storage from using DenomTrace to Denom.
func (m Migrator) MigrateDenomTraceToDenom(ctx sdk.Context) error {
	var (
		denoms      []types.Denom
		denomTraces []internaltypes.DenomTrace
	)
	m.keeper.iterateDenomTraces(ctx,
		func(dt internaltypes.DenomTrace) bool {
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

	for i := range denoms {
		m.keeper.SetDenom(ctx, denoms[i])
		m.keeper.deleteDenomTrace(ctx, denomTraces[i])
	}

	return nil
}

// setDenomTrace sets a new {trace hash -> denom trace} pair to the store.
func (k *Keeper) setDenomTrace(ctx sdk.Context, denomTrace internaltypes.DenomTrace) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.DenomTraceKey)
	bz := k.cdc.MustMarshal(&denomTrace)

	store.Set(denomTrace.Hash(), bz)
}

// deleteDenomTrace deletes the denom trace
func (k *Keeper) deleteDenomTrace(ctx sdk.Context, denomTrace internaltypes.DenomTrace) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.DenomTraceKey)
	store.Delete(denomTrace.Hash())
}

// iterateDenomTraces iterates over the denomination traces in the store
// and performs a callback function.
func (k *Keeper) iterateDenomTraces(ctx sdk.Context, cb func(denomTrace internaltypes.DenomTrace) bool) {
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
func (k *Keeper) setDenomMetadataWithDenomTrace(ctx sdk.Context, denomTrace internaltypes.DenomTrace) {
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
