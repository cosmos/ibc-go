package types

type CallbackTrigger string

const (
	ModuleName = "ibccallbacks"

	CallbackTriggerSendPacket            CallbackTrigger = "send_packet"
	CallbackTriggerAcknowledgementPacket CallbackTrigger = "acknowledgement_packet"
	CallbackTriggerTimeoutPacket         CallbackTrigger = "timeout_packet"
	CallbackTriggerReceivePacket         CallbackTrigger = "receive_packet"

	// Source callback packet data is set inside the underlying packet data using the this key.
	// ICS20 and ICS27 will store the callback packet data in the memo field as a json object.
	// The expected format is as follows:
	// {"src_callback": { ... }}
	SourceCallbackKey = "src_callback"
	// Destination callback packet data is set inside the underlying packet data using the this key.
	// ICS20 and ICS27 will store the callback packet data in the memo field as a json object.
	// The expected format is as follows:
	// {"dest_callback": { ... }}
	DestinationCallbackKey = "dest_callback"
	// Callbacks' packet data is expected to contain the callback address under this key.
	// The expected format for ICS20 and ICS27 memo field is as follows:
	// { "{callbackKey}": { "address": {stringCallbackAddress}}
	CallbackAddressKey = "address"
	// Callbacks' packet data is expected to specify the user defined gas limit under this key.
	// The expected format for ICS20 and ICS27 memo field is as follows:
	// { "{callbackKey}": { ... , "gas_limit": {stringForCallback} }
	UserDefinedGasLimitKey = "gas_limit"
)
