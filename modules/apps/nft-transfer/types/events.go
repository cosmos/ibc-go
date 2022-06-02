package types

// IBC transfer events
const (
	EventTypeTimeout      = "timeout"
	EventTypePacket       = "non_fungible_token_packet"
	EventTypeTransfer     = "ibc_nft_transfer"
	EventTypeChannelClose = "channel_closed"
	EventTypeDenomTrace   = "denomination_trace"

	AttributeKeyReceiver   = "receiver"
	AttributeKeyClassID    = "classID"
	AttributeKeyTokenIDs   = "tokenIDs"
	AttributeKeyAckSuccess = "success"
)
