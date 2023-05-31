package types

import clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"

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
