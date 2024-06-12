package types

import (
	errorsmod "cosmossdk.io/errors"

	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

const MaximumNumberOfForwardingHops = 64

// NewForwarding creates a new Forwarding instance given a memo and a variable number of hops.
func NewForwarding(memo string, hops ...Hop) *Forwarding {
	return &Forwarding{
		Memo: memo,
		Hops: hops,
	}
}

// Validate performs a basic validation of the ForwardingInfo fields.
func (fi Forwarding) Validate() error {
	if len(fi.Hops) > MaximumNumberOfForwardingHops {
		return errorsmod.Wrapf(ErrInvalidForwardingInfo, "number of hops in forwarding path cannot exceed %d", MaximumNumberOfForwardingHops)
	}

	for _, hop := range fi.Hops {
		if err := host.PortIdentifierValidator(hop.PortId); err != nil {
			return errorsmod.Wrapf(err, "invalid source port ID %s", hop.PortId)
		}
		if err := host.ChannelIdentifierValidator(hop.ChannelId); err != nil {
			return errorsmod.Wrapf(err, "invalid source channel ID %s", hop.ChannelId)
		}
	}

	if len(fi.Memo) > MaximumMemoLength {
		return errorsmod.Wrapf(ErrInvalidMemo, "memo length cannot exceed %d", MaximumMemoLength)
	}

	return nil
}
