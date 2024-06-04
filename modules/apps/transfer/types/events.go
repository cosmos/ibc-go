package types

// IBC transfer events
const (
	EventTypeTimeout      = "timeout"
	EventTypePacket       = "fungible_token_packet"
	EventTypeTransfer     = "ibc_transfer"
	EventTypeChannelClose = "channel_closed"
	EventTypeDenom        = "denomination"

	AttributeKeySender         = "sender"
	AttributeKeyReceiver       = "receiver"
	AttributeKeyDenom          = "denom"
	AttributeKeyTokens         = "tokens"
	AttributeKeyRefundReceiver = "refund_receiver"
	AttributeKeyRefundTokens   = "refund_tokens"
	AttributeKeyAckSuccess     = "success"
	AttributeKeyAck            = "acknowledgement"
	AttributeKeyAckError       = "error"
	AttributeKeyTraceHash      = "trace_hash"
	AttributeKeyMemo           = "memo"
)
