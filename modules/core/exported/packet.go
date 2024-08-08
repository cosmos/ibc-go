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

// RecvPacketStatus defines an enum type to signal the result status of an IBC application packet datagram and inform
// core IBC of how to handle the application state changes held in the cache store which it was provided for OnRecvPacket callback execution.
type RecvPacketStatus uint32

const (
	// Success instructs that the IBC application state should be persisted.
	Success RecvPacketStatus = iota
	// Failure instructs that the IBC application state should be discarded.
	Failure
	// Async instructs that the IBC application state should be persisted
	// and acknowledgement data will be written later asynchronously.
	Async
)

// String implements the fmt.Stringer interface.
func (r RecvPacketStatus) String() string {
	return [...]string{"Success", "Failure", "Async"}[r]
}

// RecvPacketResult defines a result type used to encapsulate opaque application acknowledgement data, as well as
// a status to indicate success, failure or asynchronous handling of a packet acknowledgement in the OnRecvPacket callback.
// Status is used by core IBC to ensure partial state changes are not committed
// when packet receives have not properly succeeded (typically resulting in an error acknowledgement being returned).
// During OnRecvPacket callback execution, all state changes are held in a cache store and committed if
// the result status is indicated as Async or Success.
//
// Note: IBC application callback events are always persisted so long as `RecvPacket` succeeds without error.
type RecvPacketResult struct {
	Status          RecvPacketStatus
	Acknowledgement []byte
}

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
