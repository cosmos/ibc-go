package types

import (
	"fmt"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

// NewForwardErrorAcknowledgement returns a new error acknowledgement with path forwarding information.
func NewForwardErrorAcknowledgement(packet channeltypes.Packet, ack channeltypes.Acknowledgement) channeltypes.Acknowledgement {
	ackErr := fmt.Sprintf("forwarding packet failed on %s/%s: %s",
		packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetDestPort(), packet.GetDestChannel(), ack.GetError())
	return channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{
			Error: ackErr,
		},
	}
}

// NewForwardErrorAcknowledgement returns a new error acknowledgement with path forwarding information.
func NewForwardTimeoutAcknowledgement(packet channeltypes.Packet) channeltypes.Acknowledgement {
	ackErr := fmt.Sprintf("forward packet timeout: source: %s/%s destination: %s/%s",
		packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetDestPort(), packet.GetDestChannel())
	return channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{
			Error: ackErr,
		},
	}
}
