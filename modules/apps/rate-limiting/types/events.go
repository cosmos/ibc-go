package types

var (
	EventTransferDenied = "transfer_denied"

	EventRateLimitExceeded = "rate_limit_exceeded"
	EventBlacklistedDenom  = "blacklisted_denom"

	AttributeKeyReason          = "reason"
	AttributeKeyModule          = "module"
	AttributeKeyAction          = "action"
	AttributeKeyDenom           = "denom"
	AttributeKeyChannelOrClient = "channel_or_client"
	AttributeKeyAmount          = "amount"
	AttributeKeyError           = "error"
)
