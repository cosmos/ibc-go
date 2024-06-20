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
	if len(fi.Hops) > MaximumNumberOfForwardingHops {
		return errorsmod.Wrapf(ErrInvalidForwarding, "number of hops in forwarding cannot exceed %d", MaximumNumberOfForwardingHops)
	}

	for _, hop := range fi.Hops {
		if err := host.PortIdentifierValidator(hop.PortId); err != nil {
			return errorsmod.Wrapf(err, "invalid hop source port ID %s", hop.PortId)
		}
		if err := host.ChannelIdentifierValidator(hop.ChannelId); err != nil {
			return errorsmod.Wrapf(err, "invalid source channel ID %s", hop.ChannelId)
		}
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
	if len(fi.Hops) > MaximumNumberOfForwardingHops {
		return errorsmod.Wrapf(ErrInvalidForwarding, "number of hops in forwarding packet data cannot exceed %d", MaximumNumberOfForwardingHops)
	}

	if len(fi.DestinationMemo) > MaximumMemoLength {
		return errorsmod.Wrapf(ErrInvalidMemo, "memo length cannot exceed %d", MaximumMemoLength)
	}

	if len(fi.Hops) == 0 && fi.DestinationMemo != "" {
		return errorsmod.Wrap(ErrInvalidForwarding, "memo specified when forwarding packet data hops is empty")
	}

	for _, hop := range fi.Hops {
		if err := host.PortIdentifierValidator(hop.PortId); err != nil {
			return errorsmod.Wrapf(err, "invalid source port ID %s", hop.PortId)
		}
		if err := host.ChannelIdentifierValidator(hop.ChannelId); err != nil {
			return errorsmod.Wrapf(err, "invalid source channel ID %s", hop.ChannelId)
		}
	}

	return nil
}
