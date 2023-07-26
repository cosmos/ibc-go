package types

type CallbackType string

const (
	ModuleName = "ibccallbacks"

	CallbackTypeSendPacket           CallbackType = "send_packet"
	CallbackTypeAcknowledgement      CallbackType = "acknowledgement"
	CallbackTypeTimeoutPacket        CallbackType = "timeout"
	CallbackTypeWriteAcknowledgement CallbackType = "write_acknowledgement"

	// Additional packet data is expected to specify the source callback in the following format
	// under this key:
	// {"src_callback": { ... }}
	SourceCallbackMemoKey = "src_callback"
	// Additional packet data is expected to specify the destination callback in the following format
	// under this key:
	// {"dest_callback": { ... }}
	DestCallbackMemoKey = "dest_callback"
	// Additional packet data is expected to contain the callback address in the following format:
	// { "{callbackKey}": { "address": {stringCallbackAddress}}
	CallbackAddressKey = "address"
	// Additional packet data is expected to specify the user defined gas limit in the following format:
	// { "{callbackKey}": { ... , "gas_limit": {stringForCallback} }
	UserDefinedGasLimitKey = "gas_limit"
)
