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

// PacketDataProvider defines an optional interfaces for retrieving custom packet data stored on behalf of another application.
// An existing problem in the IBC middleware design is the inability for a middleware to define its own packet data type and insert packet sender provided information.
// A short term hack was introduced into several packet data's to utilize a memo to carry this information on behalf of another application.
// This interfaces standardizes that behaviour. Upon realization of the ability for middleware's to define their own packet data types, this interface will be deprecated and removed with time.
type PacketDataProvider interface {
	// GetCustomPacketData returns the packet data held on behalf of another application.
	// The name the information is stored under should be provided as the key
	// If no custom packet data exists for the key, nil is returned.
	GetCustomPacketData(key string) interface{}
}

// PacketSenderRetriever defines an optional interface for retrieving the packet sender from packet data.
type PacketSenderRetriever interface {
	// GetPacketSender returns the sender address of the packet.
	// If the packet sender is unknown or undefined, an empty string should be returned.
	GetPacketSender(srcPortID string) string
}
