package types

// ICS27 GMP events
const (
	EventTypePacket     = "ics27_gmp_packet"
	EventTypeSendCall   = "ics27_gmp_send_call"
	EventTypeRecvPacket = "ics27_gmp_recv_packet"

	AttributeKeySender            = "sender"
	AttributeKeyReceiver          = "receiver"
	AttributeKeySalt              = "salt"
	AttributeKeyPayload           = "payload"
	AttributeKeyMemo              = "memo"
	AttributeKeySourceClient      = "source_client"
	AttributeKeyDestinationClient = "destination_client"
	AttributeKeySourcePort        = "source_port"
	AttributeKeyDestinationPort   = "destination_port"
	AttributeKeySequence          = "sequence"
	AttributeKeyAckError          = "error"
	AttributeKeyAckSuccess        = "success"
)
