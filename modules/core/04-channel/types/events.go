package types

import (
	"fmt"

	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// IBC channel events
const (
	AttributeKeyConnectionID       = "connection_id"
	AttributeKeyPortID             = "port_id"
	AttributeKeyChannelID          = "channel_id"
	AttributeKeyChannelState       = "channel_state"
	AttributeVersion               = "version"
	AttributeCounterpartyPortID    = "counterparty_port_id"
	AttributeCounterpartyChannelID = "counterparty_channel_id"

	EventTypeSendPacket        = "send_packet"
	EventTypeRecvPacket        = "recv_packet"
	EventTypeWriteAck          = "write_acknowledgement"
	EventTypeAcknowledgePacket = "acknowledge_packet"
	EventTypeTimeoutPacket     = "timeout_packet"

	// Deprecated: in favor of AttributeKeyDataHex
	AttributeKeyData = "packet_data"
	// Deprecated: in favor of AttributeKeyAckHex
	AttributeKeyAck = "packet_ack"

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

	// upgrade specific keys
	AttributeKeyUpgradeSequence       = "upgrade_sequence"
	AttributeKeyUpgradeVersion        = "upgrade_version"
	AttributeKeyUpgradeConnectionHops = "upgrade_connection_hops"
	AttributeKeyUpgradeOrdering       = "upgrade_ordering"
	AttributeKeyUpgradeErrorReceipt   = "upgrade_error_receipt"
	AttributeKeyUpgradeTimeout        = "upgrade_timeout"
)

// IBC channel events vars
var (
	EventTypeChannelOpenInit       = "channel_open_init"
	EventTypeChannelOpenTry        = "channel_open_try"
	EventTypeChannelOpenAck        = "channel_open_ack"
	EventTypeChannelOpenConfirm    = "channel_open_confirm"
	EventTypeChannelCloseInit      = "channel_close_init"
	EventTypeChannelCloseConfirm   = "channel_close_confirm"
	EventTypeChannelClosed         = "channel_close"
	EventTypeChannelUpgradeInit    = "channel_upgrade_init"
	EventTypeChannelUpgradeTry     = "channel_upgrade_try"
	EventTypeChannelUpgradeAck     = "channel_upgrade_ack"
	EventTypeChannelUpgradeConfirm = "channel_upgrade_confirm"
	EventTypeChannelUpgradeOpen    = "channel_upgrade_open"
	EventTypeChannelUpgradeTimeout = "channel_upgrade_timeout"
	EventTypeChannelUpgradeCancel  = "channel_upgrade_cancelled"
	EventTypeChannelUpgradeError   = "channel_upgrade_error"

	AttributeValueCategory = fmt.Sprintf("%s_%s", ibcexported.ModuleName, SubModuleName)
)
