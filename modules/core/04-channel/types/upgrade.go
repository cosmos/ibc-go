package types

import (
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v7/internal/collections"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

// NewUpgrade creates a new Upgrade instance.
func NewUpgrade(upgradeFields UpgradeFields, timeout Timeout, latestPacketSent uint64) Upgrade {
	return Upgrade{
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

// IsValid returns true if either the height or timestamp is non-zero
func (ut Timeout) IsValid() bool {
	return !ut.Height.IsZero() || ut.Timestamp != 0
}

// UpgradeError defines an error that occurs during an upgrade.
type UpgradeError struct {
	// err is the underlying error that caused the upgrade to fail.
	// this error should not be written to state.
	err error
	// sequence is the upgrade sequence number of the upgrade that failed.
	sequence uint64
}

// NewUpgradeError returns a new UpgradeError instance.
func NewUpgradeError(upgradeSequence uint64, err error) UpgradeError {
	return UpgradeError{
		err:      err,
		sequence: upgradeSequence,
	}
}

// Error implements the error interface, returning the underlying error which caused the upgrade to fail.
func (u UpgradeError) Error() string {
	return u.err.Error()
}

// GetErrorReceipt returns an error receipt with the code from the underlying error type stripped.
func (u UpgradeError) GetErrorReceipt() ErrorReceipt {
	// restoreErrorString defines a string constant included in error receipts.
	// NOTE: Changing this const is state machine breaking as it is written into state.
	const restoreErrorString = "restored channel to pre-upgrade state"

	_, code, _ := errorsmod.ABCIInfo(u.err, false) // discard non-determinstic codespace and log values
	return ErrorReceipt{
		Sequence: u.sequence,
		Message:  fmt.Sprintf("ABCI code: %d: %s", code, restoreErrorString),
	}
}
