package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v7/internal/collections"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

// NewUpgrade creates a new Upgrade instance.
func NewUpgrade(upgradeFields UpgradeFields, timeout Timeout, latestPacketSent uint64) *Upgrade {
	return &Upgrade{
		Fields:             upgradeFields,
		Timeout:            timeout,
		LatestSequenceSend: latestPacketSent,
	}
}

// NewUpgradeFields returns a new ModifiableUpgradeFields instance.
func NewUpgradeFields(ordering Order, connectionHops []string, version string) UpgradeFields {
	return UpgradeFields{
		Ordering:       ordering,
		ConnectionHops: connectionHops,
		Version:        version,
	}
}

// NewUpgradeTimeout returns a new UpgradeTimeout instance.
func NewUpgradeTimeout(height clienttypes.Height, timestamp uint64) Timeout {
	return Timeout{
		Height:    height,
		Timestamp: timestamp,
	}
}

// ValidateBasic performs a basic validation of the upgrade fields
func (u Upgrade) ValidateBasic() error {
	if err := u.Fields.ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "proposed upgrade fields are invalid")
	}

	if !u.Timeout.IsValid() {
		return errorsmod.Wrap(ErrInvalidUpgrade, "upgrade timeout height and upgrade timeout timestamp cannot both be 0")
	}

	return nil
}

// ValidateBasic performs a basic validation of the proposed upgrade fields
func (uf UpgradeFields) ValidateBasic() error {
	if !collections.Contains(uf.Ordering, []Order{ORDERED, UNORDERED}) {
		return errorsmod.Wrap(ErrInvalidChannelOrdering, uf.Ordering.String())
	}

	if len(uf.ConnectionHops) != 1 {
		return errorsmod.Wrap(ErrTooManyConnectionHops, "current IBC version only supports one connection hop")
	}

	if strings.TrimSpace(uf.Version) == "" {
		return errorsmod.Wrap(ErrInvalidChannelVersion, "version cannot be empty")
	}

	return nil
}

// Timeout defines an exeuction deadline structure for 04-channel msg handlers.
// This includes packet lifecycle handlers as well as handshake and upgrade protocol handlers.
// A valid Timeout contains either one or both of a timestamp and block height (sequence).

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
