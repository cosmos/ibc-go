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

// CallbackPacketData defines the interface used by ADR 008 implementations
// to obtain callback addresses associated with a specific packet data type.
// This is an optional interface which indicates support for ADR 8 implementations.
// See https://github.com/cosmos/ibc-go/tree/main/docs/architecture/adr-008-app-caller-cbs
// for more information.
type CallbackPacketData interface {
	// GetSourceCallbackAddress should return the callback address of a packet data on the source chain.
	// This may or may not be the sender of the packet. If no source callback address exists for the packet,
	// an empty string may be returned.
	GetSourceCallbackAddress() string

	// GetDestCallbackAddress should return the callback address of a packet data on the destination chain.
	// This may or may not be the receiver of the packet. If no dest callback address exists for the packet,
	// an empty string may be returned.
	GetDestCallbackAddress() string

	// GetUserDefinedCustomMessage should return a custom message defined by the sender of the packet.
	// This message may be nil if no custom message was defined.
	GetUserDefinedCustomMessage() []byte

	// UserDefinedGasLimit allows the sender of the packet to define inside the packet data
	// a gas limit for how much the ADR-8 callbacks can consume. If defined, this will be passed
	// in as the gas limit so that the callback is guaranteed to complete within a specific limit.
	// On recvPacket, a gas-overflow will just fail the transaction allowing it to timeout on the sender side.
	// On ackPacket and timeoutPacket, a gas-overflow will reject state changes made during callback but still
	// commit the transaction. This ensures the packet lifecycle can always complete.
	// If the packet data returns 0, the remaining gas limit will be passed in (modulo any chain-defined limit)
	// Otherwise, we will set the gas limit passed into the callback to the `min(ctx.GasLimit, UserDefinedGasLimit())`
	UserDefinedGasLimit() uint64
}
