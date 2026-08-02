package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (q Quota) validate() error {
	if q.MaxPercentSend.IsNil() || q.MaxPercentRecv.IsNil() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "max-percent-send and max-percent-recv must be specified")
	}

	if q.MaxPercentSend.GT(sdkmath.NewInt(100)) || q.MaxPercentSend.LT(sdkmath.ZeroInt()) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"max-percent-send percent must be between 0 and 100 (inclusively), Provided: %v", q.MaxPercentSend)
	}

	if q.MaxPercentRecv.GT(sdkmath.NewInt(100)) || q.MaxPercentRecv.LT(sdkmath.ZeroInt()) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"max-percent-recv percent must be between 0 and 100 (inclusively), Provided: %v", q.MaxPercentRecv)
	}

	if q.MaxPercentRecv.IsZero() && q.MaxPercentSend.IsZero() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"either the max send or max receive threshold must be greater than 0")
	}

	// A zero duration never resets the quota, see BeginBlocker.
	if q.DurationHours == 0 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "duration can not be zero")
	}

	return nil
}

// CheckExceedsQuota checks if new in/out flow is going to reach the max in/out or not
func (q *Quota) CheckExceedsQuota(direction PacketDirection, amount sdkmath.Int, totalValue sdkmath.Int) bool {
	// If there's no channel value (this should be almost impossible), it means there is no
	// supply of the asset, so we shouldn't prevent inflows/outflows
	if totalValue.IsZero() {
		return false
	}
	var threshold sdkmath.Int
	if direction == PACKET_RECV {
		threshold = totalValue.Mul(q.MaxPercentRecv).Quo(sdkmath.NewInt(100))
	} else {
		threshold = totalValue.Mul(q.MaxPercentSend).Quo(sdkmath.NewInt(100))
	}

	// Revert to GT check as in the original reference module
	return amount.GT(threshold)
}
