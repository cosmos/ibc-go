package types

import (
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

// NewTimeout creates a new Timeout instance.
func NewTimeout(height clienttypes.Height, timestamp uint64) Timeout {
	return Timeout{
		Height:    height,
		Timestamp: timestamp,
	}
}

// AfterHeight returns true if Timeout height is greater than the provided height.
func (t Timeout) AfterHeight(height clienttypes.Height) bool {
	return t.Height.GT(height)
}

// AfterTimestamp returns true is Timeout timestamp is greater than the provided timestamp.
func (t Timeout) AfterTimestamp(timestamp uint64) bool {
	return t.Timestamp > timestamp
}

// IsValid validates the Timeout. It ensures that either height or timestamp is set.
func (t Timeout) IsValid() bool {
	return !t.ZeroHeight() || !t.ZeroTimestamp()
}

// ZeroHeight returns true if Timeout height is zero, otherwise false.
func (t Timeout) ZeroHeight() bool {
	return t.Height.IsZero()
}

// ZeroTimestamp returns true if Timeout timestamp is zero, otherwise false.
func (t Timeout) ZeroTimestamp() bool {
	return t.Timestamp == 0
}
