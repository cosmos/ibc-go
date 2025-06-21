package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// InitGenesis initializes the rate-limiting module's state from a provided genesis state.
func (k *Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {
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

	// Set pending sequence numbers - validating that they're in right format of {channelId}/{sequenceNumber}
	for _, pendingPacketID := range state.PendingSendPacketSequenceNumbers {
		channelOrClientID, sequence, err := types.ParsePendingPacketID(pendingPacketID)
		if err != nil {
			panic(err.Error())
		}
		k.SetPendingSendPacket(ctx, channelOrClientID, sequence)
	}

	// If the hour epoch has been initialized already (epoch number != 0), validate and then use it
	if state.HourEpoch.EpochNumber > 0 {
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

	return &types.GenesisState{
		RateLimits:                       rateLimits,
		BlacklistedDenoms:                k.GetAllBlacklistedDenoms(ctx),
		WhitelistedAddressPairs:          k.GetAllWhitelistedAddressPairs(ctx),
		PendingSendPacketSequenceNumbers: k.GetAllPendingSendPackets(ctx),
		HourEpoch:                        hourEpoch,
	}
}
