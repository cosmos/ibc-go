package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	internaltypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// SetDenomTraces is a wrapper around iterateDenomTraces for testing purposes.
func (k Keeper) SetDenomTrace(ctx sdk.Context, denomTrace internaltypes.DenomTrace) {
	k.setDenomTrace(ctx, denomTrace)
}

// IterateDenomTraces is a wrapper around iterateDenomTraces for testing purposes.
func (k Keeper) IterateDenomTraces(ctx sdk.Context, cb func(denomTrace internaltypes.DenomTrace) bool) {
	k.iterateDenomTraces(ctx, cb)
}

// GetAllDenomTraces returns the trace information for all the denominations.
func (k Keeper) GetAllDenomTraces(ctx sdk.Context) []internaltypes.DenomTrace {
	var traces []internaltypes.DenomTrace
	k.iterateDenomTraces(ctx, func(denomTrace internaltypes.DenomTrace) bool {
		traces = append(traces, denomTrace)
		return false
	})

	return traces
}

// TokenFromCoin is a wrapper around tokenFromCoin for testing purposes.
func (k Keeper) TokenFromCoin(ctx sdk.Context, coin sdk.Coin) (types.Token, error) {
	return k.tokenFromCoin(ctx, coin)
}

// UnwindHops is a wrapper around unwindHops for testing purposes.
func (k Keeper) UnwindHops(ctx sdk.Context, msg *types.MsgTransfer) (*types.MsgTransfer, error) {
	return k.unwindHops(ctx, msg)
}

// GetForwardedPacket is a wrapper around getForwardedPacket for testing purposes.
func (k Keeper) GetForwardedPacket(ctx sdk.Context, portID, channelID string, sequence uint64) (channeltypes.Packet, bool) {
	return k.getForwardedPacket(ctx, portID, channelID, sequence)
}

// SetForwardedPacket is a wrapper around setForwardedPacket for testing purposes.
func (k Keeper) SetForwardedPacket(ctx sdk.Context, portID, channelID string, sequence uint64, packet channeltypes.Packet) {
	k.setForwardedPacket(ctx, portID, channelID, sequence, packet)
}

// GetAllForwardedPackets is a wrapper around getAllForwardedPackets for testing purposes.
func (k Keeper) GetAllForwardedPackets(ctx sdk.Context) []types.ForwardedPacket {
	return k.getAllForwardedPackets(ctx)
}

// IsBlockedAddr is a wrapper around isBlockedAddr for testing purposes
func (k Keeper) IsBlockedAddr(addr sdk.AccAddress) bool {
	return k.isBlockedAddr(addr)
}

// CreatePacketDataBytesFromVersion is a wrapper around createPacketDataBytesFromVersion for testing purposes
func CreatePacketDataBytesFromVersion(appVersion, sender, receiver, memo string, tokens types.Tokens, hops []types.Hop) ([]byte, error) {
	return createPacketDataBytesFromVersion(appVersion, sender, receiver, memo, tokens, hops)
}
