package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// The total value on a given path (aka, the denominator in the percentage calculation)
// is the total supply of the given denom
func (k *Keeper) GetChannelValue(ctx sdk.Context, denom string) sdkmath.Int {
	return k.bankKeeper.GetSupply(ctx, denom).Amount
}

// CheckRateLimitAndUpdateFlow checks whether the given packet will exceed the rate limit.
// Called by OnRecvPacket and OnSendPacket
func (k *Keeper) CheckRateLimitAndUpdateFlow(ctx sdk.Context, direction types.PacketDirection, packetInfo RateLimitedPacketInfo) (bool, error) {
	denom := packetInfo.Denom
	channelOrClientID := packetInfo.ChannelID
	amount := packetInfo.Amount

	// First check if the denom is blacklisted
	if k.IsDenomBlacklisted(ctx, denom) {
		err := errorsmod.Wrapf(types.ErrDenomIsBlacklisted, "denom %s is blacklisted", denom)
		EmitTransferDeniedEvent(ctx, types.EventBlacklistedDenom, denom, channelOrClientID, direction, amount, err)
		return false, err
	}

	// If there's no rate limit yet for this denom, no action is necessary
	rateLimit, found := k.GetRateLimit(ctx, denom, channelOrClientID)
	if !found {
		return false, nil
	}

	// Check if the sender/receiver pair is whitelisted
	// If so, return a success without modifying the quota
	if k.IsAddressPairWhitelisted(ctx, packetInfo.Sender, packetInfo.Receiver) {
		return false, nil
	}

	// Update the flow object with the change in amount
	if err := rateLimit.UpdateFlow(direction, amount); err != nil {
		// If the rate limit was exceeded, emit an event
		EmitTransferDeniedEvent(ctx, types.EventRateLimitExceeded, denom, channelOrClientID, direction, amount, err)
		return false, err
	}

	// If there's no quota error, update the rate limit object in the store with the new flow
	k.SetRateLimit(ctx, rateLimit)

	return true, nil
}

// If a SendPacket fails or times out, undo the outflow increment that happened during the send
func (k *Keeper) UndoSendPacket(ctx sdk.Context, channelOrClientID string, sequence uint64, denom string, amount sdkmath.Int) error {
	rateLimit, found := k.GetRateLimit(ctx, denom, channelOrClientID)
	if !found {
		return nil
	}

	// If the packet was sent during this quota, decrement the outflow
	// Otherwise, it can be ignored
	if k.CheckPacketSentDuringCurrentQuota(ctx, channelOrClientID, sequence) {
		rateLimit.Flow.Outflow = rateLimit.Flow.Outflow.Sub(amount)
		k.SetRateLimit(ctx, rateLimit)

		k.RemovePendingSendPacket(ctx, channelOrClientID, sequence)
	}

	return nil
}
