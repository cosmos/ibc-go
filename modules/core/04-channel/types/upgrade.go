package types

import (
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/internal/collections"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

// NewUpgrade creates a new Upgrade instance.
func NewUpgrade(modifiableFields ModifiableUpgradeFields, timeout UpgradeTimeout, lastPacketSent uint64) *Upgrade {
	return &Upgrade{
		UpgradeFields:  modifiableFields,
		Timeout:        timeout,
		LastPacketSent: lastPacketSent,
	}
}

// NewModifiableUpgradeFields returns a new ModifiableUpgradeFields instance.
func NewModifiableUpgradeFields(ordering Order, connectionHops []string, version string) ModifiableUpgradeFields {
	return ModifiableUpgradeFields{
		Ordering:       ordering,
		ConnectionHops: connectionHops,
		Version:        version,
	}
}

// NewUpgradeTimeout returns a new UpgradeTimeout instance.
func NewUpgradeTimeout(height clienttypes.Height, timestamp uint64) UpgradeTimeout {
	return UpgradeTimeout{
		TimeoutHeight:    height,
		TimeoutTimestamp: timestamp,
	}
}

// ValidateBasic performs a basic validation of the upgrade fields
func (u Upgrade) ValidateBasic() error {
	if err := u.UpgradeFields.ValidateBasic(); err != nil {
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
	if !collections.Contains(muf.Ordering, []Order{ORDERED, UNORDERED}) {
		return errorsmod.Wrap(ErrInvalidChannelOrdering, muf.Ordering.String())
	}

	if len(muf.ConnectionHops) != 1 {
		return errorsmod.Wrap(ErrTooManyConnectionHops, "current IBC version only supports one connection hop")
	}

	if strings.TrimSpace(muf.Version) == "" {
		return errorsmod.Wrap(ErrInvalidUpgrade, "proposed upgrade version cannot be empty")
	}

	return nil
}

// IsValid returns true if either the height or timestamp is non-zero
func (ut UpgradeTimeout) IsValid() bool {
	return !ut.TimeoutHeight.IsZero() || ut.TimeoutTimestamp != 0
}

// HasPassed returns true if the upgrade has passed the timeout height or timestamp
func (ut UpgradeTimeout) HasPassed(ctx sdk.Context) (bool, error) {

	if !ut.IsValid() {
		return true, errorsmod.Wrap(ErrInvalidUpgrade, "upgrade timeout cannot be empty")
	}

	selfHeight := clienttypes.GetSelfHeight(ctx)

	timeoutHeight := ut.TimeoutHeight
	if selfHeight.GTE(timeoutHeight) {
		return true, errorsmod.Wrapf(ErrInvalidUpgrade, "block height >= upgrade timeout height (%s >= %s)", selfHeight, timeoutHeight)
	}

	selfTime := uint64(ctx.BlockTime().UnixNano())
	timeoutTimestamp := ut.TimeoutTimestamp
	if selfTime >= timeoutTimestamp {
		return true, errorsmod.Wrapf(ErrInvalidUpgrade, "block timestamp >= upgrade timeout timestamp (%s >= %s)", ctx.BlockTime(), time.Unix(0, int64(timeoutTimestamp)))
	}

	return false, nil
}
