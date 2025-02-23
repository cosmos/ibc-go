package types

import (
	"fmt"

	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// IBC channel events
const (
	AttributeKeyConnectionID   = "connection_id"
	AttributeKeyPortID         = "port_id"
	AttributeKeyChannelID      = "channel_id"
	AttributeKeyChannelState   = "channel_state"
	AttributeKeyVersion        = "version"
	AttributeKeyConnectionHops = "connection_hops"
	AttributeKeyOrdering       = "ordering"

	AttributeCounterpartyPortID    = "counterparty_port_id"
	AttributeCounterpartyChannelID = "counterparty_channel_id"

	EventTypeSendPacket        = "send_packet"
	EventTypeRecvPacket        = "recv_packet"
	EventTypeWriteAck          = "write_acknowledgement"
	EventTypeAcknowledgePacket = "acknowledge_packet"
	EventTypeTimeoutPacket     = "timeout_packet"

	AttributeKeyDataHex          = "packet_data_hex"
	AttributeKeyAckHex           = "packet_ack_hex"
	AttributeKeyTimeoutHeight    = "packet_timeout_height"
	AttributeKeyTimeoutTimestamp = "packet_timeout_timestamp"
	AttributeKeySequence         = "packet_sequence"
	AttributeKeySrcPort          = "packet_src_port"
	AttributeKeySrcChannel       = "packet_src_channel"
	AttributeKeyDstPort          = "packet_dst_port"
	AttributeKeyDstChannel       = "packet_dst_channel"
	AttributeKeyChannelOrdering  = "packet_channel_ordering"
	AttributeKeyConnection       = "packet_connection"
)

// IBC channel events vars
var (
	EventTypeChannelOpenInit     = "channel_open_init"
	EventTypeChannelOpenTry      = "channel_open_try"
	EventTypeChannelOpenAck      = "channel_open_ack"
	EventTypeChannelOpenConfirm  = "channel_open_confirm"
	EventTypeChannelCloseInit    = "channel_close_init"
	EventTypeChannelCloseConfirm = "channel_close_confirm"
	EventTypeChannelClosed       = "channel_close"

	AttributeValueCategory = fmt.Sprintf("%s_%s", ibcexported.ModuleName, SubModuleName)
)
