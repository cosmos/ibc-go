package types

const (
	EventTypeIFTBridgeRegistered  = "ift_bridge_registered"
	EventTypeIFTBridgeUpdated     = "ift_bridge_updated"
	EventTypeIFTBridgeRemoved     = "ift_bridge_removed"
	EventTypeIFTTransferInitiated = "ift_transfer_initiated"
	EventTypeIFTMintReceived      = "ift_mint_received"
	EventTypeIFTTransferCompleted = "ift_transfer_completed"
	EventTypeIFTTransferRefunded  = "ift_transfer_refunded"
	EventTypeIFTCallbackFailed    = "ift_callback_failed"

	AttributeKeyDenom                  = "denom"
	AttributeKeyClientID               = "client_id"
	AttributeKeyCounterpartyIFTAddress = "counterparty_ift_address"
	AttributeKeyIFTSendCallConstructor = "ift_send_call_constructor"
	AttributeKeySequence               = "sequence"
	AttributeKeySender                 = "sender"
	AttributeKeyReceiver               = "receiver"
	AttributeKeyAmount                 = "amount"
	AttributeKeyError                  = "error"
	AttributeKeyCallbackType           = "callback_type"
)
