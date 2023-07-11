package types

type CallbackType string

const (
	ModuleName = "ibccallbacks"

	CallbackTypeAcknowledgement CallbackType = "acknowledgement"
	CallbackTypeTimeoutPacket   CallbackType = "timeout"
	CallbackTypeReceivePacket   CallbackType = "receive_packet"
)
