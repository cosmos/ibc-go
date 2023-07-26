package types

type CallbackType string

const (
	ModuleName = "ibccallbacks"

	CallbackTypeSendPacket           CallbackType = "send_packet"
	CallbackTypeAcknowledgement      CallbackType = "acknowledgement"
	CallbackTypeTimeoutPacket        CallbackType = "timeout"
	CallbackTypeWriteAcknowledgement CallbackType = "write_acknowledgement"

	SourceCallbackMemoKey = "src_callback"
	DestCallbackMemoKey   = "dest_callback"
	// The memo is expected to specify the user defined gas limit in the following format:
	// { "{callbackKey}": { ... , "gas_limit": {stringForCallback} }
	UserDefinedGasLimitKey = "gas_limit"
	// The memo is expected to contain the callback address in the following format:
	// { "{callbackKey}": { "address": {stringCallbackAddress}}
	CallbackAddressKey = "address"
)
