package types

import (
	"fmt"

	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// IBC channel events
const (
	EventTypeCreateChannel        = "create_channel"
	EventTypeRegisterCounterparty = "register_counterparty"
	EventTypeSendPacket           = "send_packet"
	EventTypeRecvPacket           = "recv_packet"
	EventTypeTimeoutPacket        = "timeout_packet"
	EventTypeAcknowledgePacket    = "acknowledge_packet"
	EventTypeWriteAck             = "write_acknowledgement"

	AttributeKeyChannelID             = "channel_id"
	AttributeKeyClientID              = "client_id"
	AttributeKeyCounterpartyChannelID = "counterparty_channel_id"
	AttributeKeySrcChannel            = "packet_source_channel"
	AttributeKeyDstChannel            = "packet_dest_channel"
	AttributeKeySequence              = "packet_sequence"
	AttributeKeyTimeoutTimestamp      = "packet_timeout_timestamp"
	AttributeKeyPacketDataHex         = "packet_data_hex"
	AttributeKeyAckDataHex            = "acknowledgement_data_hex"
)

// IBC channel events vars
var (
	AttributeValueCategory = fmt.Sprintf("%s_%s", ibcexported.ModuleName, SubModuleName)
)
