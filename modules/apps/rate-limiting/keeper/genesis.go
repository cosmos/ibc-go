package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// InitGenesis initializes the rate-limiting module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {
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
	for _, pendingPacketId := range state.PendingSendPacketSequenceNumbers {
		channelOrClientId, sequence, err := types.ParsePendingPacketId(pendingPacketId)
		if err != nil {
			panic(err.Error())
		}
		k.SetPendingSendPacket(ctx, channelOrClientId, sequence)
	}

	// If the hour epoch has been initialized already (epoch number != 0), validate and then use it
	if state.HourEpoch.EpochNumber > 0 {
		k.SetHourEpoch(ctx, state.HourEpoch)
	} else {
		// If the hour epoch has not been initialized yet, set it so that the epoch number matches
		// the current hour and the start time is precisely on the hour
		state.HourEpoch.EpochNumber = uint64(ctx.BlockTime().Hour()) //nolint:gosec
		state.HourEpoch.EpochStartTime = ctx.BlockTime().Truncate(time.Hour)
		state.HourEpoch.EpochStartHeight = ctx.BlockHeight()
		k.SetHourEpoch(ctx, state.HourEpoch)
	}
}

// ExportGenesis returns the rate-limiting module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	// params := k.GetParams(ctx)
	rateLimits := k.GetAllRateLimits(ctx)

	return &types.GenesisState{
		// Params:     params,
		RateLimits:                       rateLimits,
		BlacklistedDenoms:                k.GetAllBlacklistedDenoms(ctx),
		WhitelistedAddressPairs:          k.GetAllWhitelistedAddressPairs(ctx),
		PendingSendPacketSequenceNumbers: k.GetAllPendingSendPackets(ctx),
		HourEpoch:                        k.GetHourEpoch(ctx),
	}
}
