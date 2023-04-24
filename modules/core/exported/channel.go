package exported

// ChannelI defines the standard interface for a channel end.
type ChannelI interface {
	GetState() int32
	GetOrdering() int32
	GetCounterparty() CounterpartyChannelI
	GetConnectionHops() []string
	GetVersion() string
	ValidateBasic() error
}

// CounterpartyChannelI defines the standard interface for a channel end's
// counterparty.
type CounterpartyChannelI interface {
	GetPortID() string
	GetChannelID() string
	ValidateBasic() error
}

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
	// Success tells ibc-go if the state should be persisted. During the "OnRecvPacket" callback execution, all state is hold in a cache store and committed later. Note: events are always persisted.
	// This is independent of application level success/error which is encoded in the acknowledgement
	// bytes in a protocol specific way.
	// See https://github.com/cosmos/ibc-go/blob/main/docs/ibc/apps.md for further explanations.
	Success() bool
	Acknowledgement() []byte
}
