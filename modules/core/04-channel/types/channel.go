package types

import (
	"slices"

	errorsmod "cosmossdk.io/errors"

	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// NewChannel creates a new Channel instance
func NewChannel(
	state State, ordering Order, counterparty Counterparty,
	hops []string, version string,
) Channel {
	return Channel{
		State:          state,
		Ordering:       ordering,
		Counterparty:   counterparty,
		ConnectionHops: hops,
		Version:        version,
	}
}

// ValidateBasic performs a basic validation of the channel fields
func (ch Channel) ValidateBasic() error {
	if ch.State == UNINITIALIZED {
		return ErrInvalidChannelState
	}
	if !slices.Contains([]Order{ORDERED, UNORDERED}, ch.Ordering) {
		return errorsmod.Wrap(ErrInvalidChannelOrdering, ch.Ordering.String())
	}
	if len(ch.ConnectionHops) != 1 {
		return errorsmod.Wrap(
			ErrTooManyConnectionHops,
			"current IBC version only supports one connection hop",
		)
	}
	if err := host.ConnectionIdentifierValidator(ch.ConnectionHops[0]); err != nil {
		return errorsmod.Wrap(err, "invalid connection hop ID")
	}
	return ch.Counterparty.ValidateBasic()
}

// NewCounterparty returns a new Counterparty instance
func NewCounterparty(portID, channelID string) Counterparty {
	return Counterparty{
		PortId:    portID,
		ChannelId: channelID,
	}
}

// ValidateBasic performs a basic validation check of the identifiers
func (c Counterparty) ValidateBasic() error {
	if err := host.PortIdentifierValidator(c.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid counterparty port ID")
	}
	if c.ChannelId != "" {
		if err := host.ChannelIdentifierValidator(c.ChannelId); err != nil {
			return errorsmod.Wrap(err, "invalid counterparty channel ID")
		}
	}
	return nil
}

// NewIdentifiedChannel creates a new IdentifiedChannel instance
func NewIdentifiedChannel(portID, channelID string, ch Channel) IdentifiedChannel {
	return IdentifiedChannel{
		State:          ch.State,
		Ordering:       ch.Ordering,
		Counterparty:   ch.Counterparty,
		ConnectionHops: ch.ConnectionHops,
		Version:        ch.Version,
		PortId:         portID,
		ChannelId:      channelID,
	}
}

// ValidateBasic performs a basic validation of the identifiers and channel fields.
func (ic IdentifiedChannel) ValidateBasic() error {
	if err := host.ChannelIdentifierValidator(ic.ChannelId); err != nil {
		return errorsmod.Wrap(err, "invalid channel ID")
	}
	if err := host.PortIdentifierValidator(ic.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}
	channel := NewChannel(ic.State, ic.Ordering, ic.Counterparty, ic.ConnectionHops, ic.Version)
	return channel.ValidateBasic()
}
