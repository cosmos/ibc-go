package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
)

const MaximumNumberOfForwardingHops = 8 // denotes the maximum number of forwarding hops allowed

// NewForwarding creates a new Forwarding instance given an unwind value and a variable number of hops.
func NewForwarding(unwind bool, hops ...Hop) *Forwarding {
	return &Forwarding{
		Unwind: unwind,
		Hops:   hops,
	}
}

// Validate performs a basic validation of the Forwarding fields.
func (f Forwarding) Validate() error {
	if err := validateHops(f.GetHops()); err != nil {
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
func (fpd ForwardingPacketData) Validate() error {
	if err := validateHops(fpd.Hops); err != nil {
		return errorsmod.Wrapf(ErrInvalidForwarding, "invalid hops in forwarding packet data")
	}

	if len(fpd.DestinationMemo) > MaximumMemoLength {
		return errorsmod.Wrapf(ErrInvalidMemo, "memo length cannot exceed %d", MaximumMemoLength)
	}

	if len(fpd.Hops) == 0 && fpd.DestinationMemo != "" {
		return errorsmod.Wrap(ErrInvalidForwarding, "memo specified when forwarding packet data hops is empty")
	}

	return nil
}

// NewHop creates a Hop with the given port ID and channel ID.
func NewHop(portID, channelID string) Hop {
	return Hop{portID, channelID}
}

// Validate performs a basic validation of the Hop fields.
func (h Hop) Validate() error {
	if err := host.PortIdentifierValidator(h.PortId); err != nil {
		return errorsmod.Wrapf(err, "invalid hop source port ID %s", h.PortId)
	}
	if err := host.ChannelIdentifierValidator(h.ChannelId); err != nil {
		return errorsmod.Wrapf(err, "invalid hop source channel ID %s", h.ChannelId)
	}

	return nil
}

// String returns the Hop in the format:
// <portID>/<channelID>
func (h Hop) String() string {
	return fmt.Sprintf("%s/%s", h.PortId, h.ChannelId)
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
