package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	internaltypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/internal/types"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

// SetDenomTrace is a wrapper around setDenomTrace for testing purposes.
func (k *Keeper) SetDenomTrace(ctx sdk.Context, denomTrace internaltypes.DenomTrace) {
	k.setDenomTrace(ctx, denomTrace)
}

// IterateDenomTraces is a wrapper around iterateDenomTraces for testing purposes.
func (k *Keeper) IterateDenomTraces(ctx sdk.Context, cb func(denomTrace internaltypes.DenomTrace) bool) {
	k.iterateDenomTraces(ctx, cb)
}

// GetAllDenomTraces returns the trace information for all the denominations.
func (k *Keeper) GetAllDenomTraces(ctx sdk.Context) []internaltypes.DenomTrace {
	var traces []internaltypes.DenomTrace
	k.iterateDenomTraces(ctx, func(denomTrace internaltypes.DenomTrace) bool {
		traces = append(traces, denomTrace)
		return false
	})

	return traces
}

// CreatePacketDataBytesFromVersion is a wrapper around createPacketDataBytesFromVersion for testing purposes
func CreatePacketDataBytesFromVersion(appVersion, sender, receiver, memo string, token types.Token) ([]byte, error) {
	return createPacketDataBytesFromVersion(appVersion, sender, receiver, memo, token)
}
