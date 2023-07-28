package exported

// PacketI defines the standard interface for IBC packets
type PacketI interface {
	GetSequence() uint64
	GetTimeoutHeight() Height
	GetTimeoutTimestamp() uint64
	GetSourcePort() string
	GetSourceChannel() string
	GetDestPort() string
	GetDestChannel() string
	GetData() []byte
	ValidateBasic() error
}

// Acknowledgement defines the interface used to return
// acknowledgements in the OnRecvPacket callback.
type Acknowledgement interface {
	Success() bool
	Acknowledgement() []byte
}

// AdditionalPacketDataProvider defines the standard interface for retrieving additional packet data.
// The interface is used to retrieve json encoded data from the packet memo.
// The interface is also used to retrieve the sender address of the packet.
type AdditionalPacketDataProvider interface {
	// GetAdditionalData returns additional packet data keyed by a string.
	// This function is used to retrieve json encoded data from the packet memo.
	// If no additional data exists for the key, nil should be returned.
	GetAdditionalData(key string) interface{}
	// GetPacketSender returns the sender address of the packet.
	// If the packet sender is unknown or undefined, an empty string should be returned.
	GetPacketSender(srcPortID string) string
}
