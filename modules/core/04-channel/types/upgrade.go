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
func NewUpgrade(upgradeFields UpgradeFields, timeout UpgradeTimeout, latestPacketSent uint64) *Upgrade {
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
func NewUpgradeTimeout(height clienttypes.Height, timestamp uint64) UpgradeTimeout {
	return UpgradeTimeout{
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
func (ut UpgradeTimeout) IsValid() bool {
	return !ut.Height.IsZero() || ut.Timestamp != 0
}

// HasPassed returns true if the upgrade has passed the timeout height or timestamp
func (ut UpgradeTimeout) HasPassed(ctx sdk.Context) (bool, error) {
	if !ut.IsValid() {
		return true, errorsmod.Wrap(ErrInvalidUpgrade, "upgrade timeout cannot be empty")
	}

	selfHeight, timeoutHeight := clienttypes.GetSelfHeight(ctx), ut.Height
	if selfHeight.GTE(timeoutHeight) && timeoutHeight.GT(clienttypes.ZeroHeight()) {
		return true, errorsmod.Wrapf(ErrInvalidUpgrade, "block height >= upgrade timeout height (%s >= %s)", selfHeight, timeoutHeight)
	}

	selfTime, timeoutTimestamp := uint64(ctx.BlockTime().UnixNano()), ut.Timestamp
	if selfTime >= timeoutTimestamp && timeoutTimestamp > 0 {
		return true, errorsmod.Wrapf(ErrInvalidUpgrade, "block timestamp >= upgrade timeout timestamp (%s >= %s)", ctx.BlockTime(), time.Unix(0, int64(timeoutTimestamp)))
	}

	return false, nil
}
