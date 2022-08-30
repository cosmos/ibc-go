package types

// ICQ Interchain Query events
const (
	EventTypePacketError = "icq_packet_error"
	EventTypeQuery       = "icq_query"

	AttributeKeyAckError      = "error"
	AttributeKeyHostChannelID = "host_channel_id"
	AttributeKeyNumRequests   = "num_requests"
)
