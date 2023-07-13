package types

type CallbackType string

const (
	ModuleName = "ibccallbacks"

	CallbackTypeSendPacket      CallbackType = "send_packet"
	CallbackTypeAcknowledgement CallbackType = "acknowledgement"
	CallbackTypeTimeoutPacket   CallbackType = "timeout"
	CallbackTypeReceivePacket   CallbackType = "receive_packet"
)
