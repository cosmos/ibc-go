package types

import (
	errorsmod "cosmossdk.io/errors"

	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

const MaximumNumberOfForwardingHops = 16 // denotes the maximum number of forwarding hops allowed

// NewForwarding creates a new Forwarding instance given an unwind value and a variable number of hops.
func NewForwarding(unwind bool, hops ...Hop) Forwarding {
	return Forwarding{
		Unwind: unwind,
		Hops:   hops,
	}
}

// Validate performs a basic validation of the Forwarding fields.
func (fi Forwarding) Validate() error {
	if err := validateHops(fi.Hops); err != nil {
		return errorsmod.Wrapf(ErrInvalidForwarding, "invalid hops in forwarding")
	}

	return nil
}

// NewForwardingPacketData creates a new ForwardingPacketData instance given a memo and a variable number of hops.
func NewForwardingPacketData(destinationMemo string, hops ...Hop) ForwardingPacketData {
	return ForwardingPacketData{
		DestinationMemo: destinationMemo,
		Hops:            hops,
	}
}

// Validate performs a basic validation of the ForwardingPacketData fields.
func (fi ForwardingPacketData) Validate() error {
	if err := validateHops(fi.Hops); err != nil {
		return errorsmod.Wrapf(ErrInvalidForwarding, "invalid hops in forwarding packet data")
	}

	if len(fi.DestinationMemo) > MaximumMemoLength {
		return errorsmod.Wrapf(ErrInvalidMemo, "memo length cannot exceed %d", MaximumMemoLength)
	}

	if len(fi.Hops) == 0 && fi.DestinationMemo != "" {
		return errorsmod.Wrap(ErrInvalidForwarding, "memo specified when forwarding packet data hops is empty")
	}

	return nil
}

// Validate performs a basic validation of the Hop fields.
func (h Hop) Validate() error {
	if err := host.PortIdentifierValidator(h.PortId); err != nil {
		return errorsmod.Wrapf(err, "invalid hop source port ID %s", h.PortId)
	}
	if err := host.ChannelIdentifierValidator(h.ChannelId); err != nil {
		return errorsmod.Wrapf(err, "invalid source channel ID %s", h.ChannelId)
	}

	return nil
}

// validateHops performs a basic validation of the hops.
// It checks that the number of hops does not exceed the maximum allowed and that each hop is valid.
// It will not return any errors if hops is empty.
func validateHops(hops []Hop) error {
	if len(hops) > MaximumNumberOfForwardingHops {
		return errorsmod.Wrapf(ErrInvalidForwarding, "number of hops cannot exceed %d", MaximumNumberOfForwardingHops)
	}

	for _, hop := range hops {
		if err := hop.Validate(); err != nil {
			return err
		}
	}

	return nil
}
