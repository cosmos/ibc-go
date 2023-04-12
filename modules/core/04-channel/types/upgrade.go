package types

import (
	errorsmod "cosmossdk.io/errors"
)

// ValidateBasic performs a basic validation of the upgrade fields
func (u Upgrade) ValidateBasic() error {
	if err := u.ProposedUpgrade.ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "proposed upgrade fields are invalid")
	}

	if !u.Timeout.IsValid() {
		return errorsmod.Wrap(ErrInvalidUpgrade, "upgrade timeout cannot be empty")
	}

	// TODO: determine if last packet sequence sent can be 0?
	return nil
}

// ValidateBasic performs a basic validation of the proposed upgrade fields
func (muf ModifiableUpgradeFields) ValidateBasic() error {
	if !(muf.Ordering == ORDERED || muf.Ordering == UNORDERED) {
		return errorsmod.Wrap(ErrInvalidChannelOrdering, muf.Ordering.String())
	}
	if len(muf.ConnectionHops) != 1 {
		return errorsmod.Wrap(ErrTooManyConnectionHops, "current IBC version only supports one connection hop")
	}

	if muf.Version == "" {
		errorsmod.Wrap(ErrInvalidUpgrade, "proposed upgrade version cannot be empty")
	}

	return nil
}

// IsValid returns true if either the height or timestamp is non-zero
func (ut UpgradeTimeout) IsValid() bool {
	return !ut.TimeoutHeight.IsZero() || ut.TimeoutTimestamp != 0
}
