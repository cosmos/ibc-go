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

// RecvPacketResult ...
type RecvPacketResult struct {
	Status          RecvPacketStatus
	Acknowledgement []byte
}

// RecvPacketStatus ...
type RecvPacketStatus uint32

const (
	SUCCESS RecvPacketStatus = iota
	FAILURE
	ASYNC
)

// PacketData defines an optional interface which an application's packet data structure may implement.
type PacketData interface {
	// GetPacketSender returns the sender address of the packet data.
	// If the packet sender is unknown or undefined, an empty string should be returned.
	GetPacketSender(sourcePortID string) string
}

// PacketDataProvider defines an optional interfaces for retrieving custom packet data stored on behalf of another application.
// An existing problem in the IBC middleware design is the inability for a middleware to define its own packet data type and insert packet sender provided information.
// A short term solution was introduced into several application's packet data to utilize a memo field to carry this information on behalf of another application.
// This interfaces standardizes that behaviour. Upon realization of the ability for middleware's to define their own packet data types, this interface will be deprecated and removed with time.
type PacketDataProvider interface {
	// GetCustomPacketData returns the packet data held on behalf of another application.
	// The name the information is stored under should be provided as the key.
	// If no custom packet data exists for the key, nil should be returned.
	GetCustomPacketData(key string) interface{}
}
