package types

import (
	"time"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
)

// NewTimeout returns a new Timeout instance.
func NewTimeout(height clienttypes.Height, timestamp uint64) Timeout {
	return Timeout{
		Height:    height,
		Timestamp: timestamp,
	}
}

// IsValid returns true if either the height or timestamp is non-zero
func (t Timeout) IsValid() bool {
	return !t.Height.IsZero() || t.Timestamp != 0
}

// TODO: Update after https://github.com/cosmos/ibc-go/issues/3483 has been resolved
// HasPassed returns true if the upgrade has passed the timeout height or timestamp
func (t Timeout) HasPassed(ctx sdk.Context) (bool, error) {
	if !t.IsValid() {
		return true, errorsmod.Wrap(ErrInvalidUpgrade, "upgrade timeout cannot be empty")
	}

	selfHeight, timeoutHeight := clienttypes.GetSelfHeight(ctx), t.Height
	if selfHeight.GTE(timeoutHeight) && timeoutHeight.GT(clienttypes.ZeroHeight()) {
		return true, errorsmod.Wrapf(ErrInvalidUpgrade, "block height >= upgrade timeout height (%s >= %s)", selfHeight, timeoutHeight)
	}

	selfTime, timeoutTimestamp := uint64(ctx.BlockTime().UnixNano()), t.Timestamp
	if selfTime >= timeoutTimestamp && timeoutTimestamp > 0 {
		return true, errorsmod.Wrapf(ErrInvalidUpgrade, "block timestamp >= upgrade timeout timestamp (%s >= %s)", ctx.BlockTime(), time.Unix(0, int64(timeoutTimestamp)))
	}

	return false, nil
}
