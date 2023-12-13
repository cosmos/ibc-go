package types

import (
	errorsmod "cosmossdk.io/errors"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
)

// NewTimeout returns a new Timeout instance.
func NewTimeout(height clienttypes.Height, timestamp uint64) Timeout {
	return Timeout{
		Height:    height,
		Timestamp: timestamp,
	}
}

// IsValid returns true if either the height or timestamp is non-zero.
func (t Timeout) IsValid() bool {
	return !t.Height.IsZero() || t.Timestamp != 0
}

// Elapsed returns true if either the provided height or timestamp is past the
// respective absolute timeout values.
func (t Timeout) Elapsed(height clienttypes.Height, timestamp uint64) bool {
	return t.heightElapsed(height) || t.timestampElapsed(timestamp)
}

func (t Timeout) ErrTimeoutNotReached(height clienttypes.Height, timestamp uint64) error {
	if !t.heightElapsed(height) {
		return errorsmod.Wrapf(ErrTimeoutNotReached, "current height: %s, timeout height %s", height, t.Height)
	}

	return errorsmod.Wrapf(ErrTimeoutNotReached, "current timestamp: %d, timeout timestamp %d", timestamp, t.Timestamp)
}

func (t Timeout) ErrTimeoutElapsed(height clienttypes.Height, timestamp uint64) error {
	if !t.heightElapsed(height) {
		return errorsmod.Wrapf(ErrTimeoutElapsed, "current height: %s, timeout height %s", height, t.Height)
	}

	return errorsmod.Wrapf(ErrTimeoutElapsed, "current timestamp: %d, timeout timestamp %d", timestamp, t.Timestamp)
}

func (t Timeout) heightElapsed(height clienttypes.Height) bool {
	return !t.Height.IsZero() && height.GTE(t.Height)
}

func (t Timeout) timestampElapsed(timestamp uint64) bool {
	return t.Timestamp != 0 && timestamp >= t.Timestamp
}
