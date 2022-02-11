package types

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewIdentifiedPacketFees creates and returns a new IdentifiedPacketFees struct
func NewIdentifiedPacketFees(packetFees ...IdentifiedPacketFee) IdentifiedPacketFees {
	return IdentifiedPacketFees{
		PacketFees: packetFees,
	}
}

// Total returns the total escrowable amount for a given Fee
func (f Fee) Total() sdk.Coins {
	return f.RecvFee.Add(f.AckFee...).Add(f.TimeoutFee...)
}

// Validate asserts that each Fee is valid and all three Fees are not empty or zero
func (fee Fee) Validate() error {
	var errFees []string
	if !fee.AckFee.IsValid() {
		errFees = append(errFees, "ack fee invalid")
	}
	if !fee.RecvFee.IsValid() {
		errFees = append(errFees, "recv fee invalid")
	}
	if !fee.TimeoutFee.IsValid() {
		errFees = append(errFees, "timeout fee invalid")
	}

	if len(errFees) > 0 {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "contains invalid fees: %s", strings.Join(errFees, " , "))
	}

	// if all three fee's are zero or empty return an error
	if fee.AckFee.IsZero() && fee.RecvFee.IsZero() && fee.TimeoutFee.IsZero() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, "all fees are zero")
	}

	return nil
}
