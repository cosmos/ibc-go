package types

// CCV events
const (
	EventTypeTimeout      = "timeout"
	EventTypePacket       = "ccv_packet"
	EventTypeChannelClose = "channel_closed"

	AttributeKeyAckSuccess = "success"
	AttributeKeyAck        = "acknowledgement"
	AttributeKeyAckError   = "error"
)
