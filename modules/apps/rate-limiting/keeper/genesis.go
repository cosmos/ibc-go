package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

// InitGenesis initializes the rate-limiting module's state from a provided genesis state.
func (k *Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {
	if err := state.Validate(); err != nil {
		panic(err)
	}

	// Set rate limits, blacklists, and whitelists
	for _, rateLimit := range state.RateLimits {
		k.SetRateLimit(ctx, rateLimit)
	}
	for _, denom := range state.BlacklistedDenoms {
		k.AddDenomToBlacklist(ctx, denom)
	}
	for _, addressPair := range state.WhitelistedAddressPairs {
		k.SetWhitelistedAddressPair(ctx, addressPair)
	}

	// Set pending sequence numbers - validating that they're in right format of {channelId}/{sequenceNumber}/{denom}
	for _, pendingPacketID := range state.PendingSendPacketSequenceNumbers {
		channelOrClientID, sequence, denom, err := types.ParsePendingPacketID(pendingPacketID)
		if err != nil {
			if types.IsLegacyPendingPacketID(pendingPacketID) {
				continue
			}
			panic(err.Error())
		}
		if err := k.SetPendingSendPacket(ctx, channelOrClientID, sequence, denom); err != nil {
			panic(err)
		}
	}
	for _, pendingPacketID := range state.PendingRecvPacketSequenceNumbers {
		channelOrClientID, sequence, denom, err := types.ParsePendingPacketID(pendingPacketID)
		if err != nil {
			if types.IsLegacyPendingPacketID(pendingPacketID) {
				continue
			}
			panic(err.Error())
		}
		if err := k.SetPendingReceivePacket(ctx, channelOrClientID, sequence, denom); err != nil {
			panic(err)
		}
	}

	// The epoch is initialized iff its start time is set. EpochNumber cannot be
	// the discriminator: 0 is also the legitimate number seeded for a chain
	// whose genesis time falls in the 00:00 UTC hour.
	if !state.HourEpoch.EpochStartTime.IsZero() {
		if err := k.SetHourEpoch(ctx, state.HourEpoch); err != nil {
			panic(err)
		}
	} else {
		// If the hour epoch has not been initialized yet, set it so that the epoch number matches
		// the current hour and the start time is precisely on the hour
		state.HourEpoch.EpochNumber = uint64(ctx.BlockTime().Hour()) //nolint:gosec
		state.HourEpoch.EpochStartTime = ctx.BlockTime().Truncate(time.Hour)
		state.HourEpoch.EpochStartHeight = ctx.BlockHeight()
		if err := k.SetHourEpoch(ctx, state.HourEpoch); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the rate-limiting module's exported genesis.
func (k *Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	rateLimits := k.GetAllRateLimits(ctx)
	hourEpoch, err := k.GetHourEpoch(ctx)
	if err != nil {
		panic(err)
	}

	pendingSendPackets, err := k.GetAllPendingSendPackets(ctx)
	if err != nil {
		panic(err)
	}
	pendingReceivePackets, err := k.GetAllPendingReceivePackets(ctx)
	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		RateLimits:                       rateLimits,
		BlacklistedDenoms:                k.GetAllBlacklistedDenoms(ctx),
		WhitelistedAddressPairs:          k.GetAllWhitelistedAddressPairs(ctx),
		PendingSendPacketSequenceNumbers: pendingSendPackets,
		PendingRecvPacketSequenceNumbers: pendingReceivePackets,
		HourEpoch:                        hourEpoch,
	}
}
