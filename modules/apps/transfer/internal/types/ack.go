package types

import (
	"fmt"

	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// NewForwardErrorAcknowledgement returns a new error acknowledgement with path forwarding information.
func NewForwardErrorAcknowledgement(sourcePort, sourceChannel string, ack channeltypes.Acknowledgement) channeltypes.Acknowledgement {
	ackErr := fmt.Sprintf("forwarding packet failed on %s/%s: %s", sourcePort, sourceChannel, ack.GetError())
	return channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{
			Error: ackErr,
		},
	}
}

// NewForwardTimeoutAcknowledgement returns a new error acknowledgement with path forwarding information.
func NewForwardTimeoutAcknowledgement(sourcePort, sourceChannel string) channeltypes.Acknowledgement {
	ackErr := fmt.Sprintf("forwarding packet timed out on %s/%s", sourcePort, sourceChannel)
	return channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{
			Error: ackErr,
		},
	}
}
