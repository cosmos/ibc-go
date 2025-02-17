package types

import (
	fmt "fmt"

	errorsmod "cosmossdk.io/errors"

	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

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
