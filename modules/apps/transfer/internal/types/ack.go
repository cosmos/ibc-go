package types

import (
	"fmt"

	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// NewForwardErrorAcknowledgement returns a new error acknowledgement with path forwarding information.
func NewForwardErrorAcknowledgement(packet channeltypes.Packet, ack channeltypes.Acknowledgement) channeltypes.Acknowledgement {
	ackErr := fmt.Sprintf("forwarding packet failed on %s/%s: %s", packet.GetSourcePort(), packet.GetSourceChannel(), ack.GetError())
	return channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{
			Error: ackErr,
		},
	}
}

// NewForwardTimeoutAcknowledgement returns a new error acknowledgement with path forwarding information.
func NewForwardTimeoutAcknowledgement(packet channeltypes.Packet) channeltypes.Acknowledgement {
	ackErr := fmt.Sprintf("forwarding packet timed out on %s/%s", packet.GetSourcePort(), packet.GetSourceChannel())
	return channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{
			Error: ackErr,
		},
	}
}
