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
	EventTypeWriteAcknowledgement = "write_acknowledgement"

	EventTypeSendPayload = "send_payload"

	AttributeKeyChannelID             = "channel_id"
	AttributeKeyClientID              = "client_id"
	AttributeKeyCounterpartyChannelID = "counterparty_channel_id"
	AttributeKeySrcChannel            = "packet_source_channel"
	AttributeKeyDstChannel            = "packet_dest_channel"
	AttributeKeySequence              = "packet_sequence"
	AttributeKeyTimeoutTimestamp      = "packet_timeout_timestamp"
	AttributeKeyPayloadLength         = "packet_payload_length"
	AttributeKeyPayloadSequence       = "payload_sequence"
	AttributeKeyVersion               = "payload_version"
	AttributeKeyEncoding              = "payload_encoding"
	AttributeKeyData                  = "payload_data"
	AttributeKeyAcknowledgement       = "acknowledgement"
)

// IBC channel events vars
var (
	AttributeValueCategory = fmt.Sprintf("%s_%s", ibcexported.ModuleName, SubModuleName)
)
