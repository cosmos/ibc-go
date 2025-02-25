package types

import (
	"fmt"

	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// IBC v2 core events
const (
	EventTypeSendPacket        = "send_packet"
	EventTypeRecvPacket        = "recv_packet"
	EventTypeTimeoutPacket     = "timeout_packet"
	EventTypeAcknowledgePacket = "acknowledge_packet"
	EventTypeWriteAck          = "write_acknowledgement"

	AttributeKeySrcClient        = "packet_source_client"
	AttributeKeyDstClient        = "packet_dest_client"
	AttributeKeySequence         = "packet_sequence"
	AttributeKeyTimeoutTimestamp = "packet_timeout_timestamp"
	AttributeKeyEncodedPacketHex = "encoded_packet_hex"
	AttributeKeyEncodedAckHex    = "encoded_acknowledgement_hex"
)

// IBC v2 core events vars
var (
	AttributeValueCategory = fmt.Sprintf("%s_%s", ibcexported.ModuleName, SubModuleName)
)
