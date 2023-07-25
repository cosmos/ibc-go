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

	// GetSourceUserDefinedGasLimit allows the sender of the packet to define the minimum amount of gas that the
	// relayer must set for the source callback executions. If this value is greater than the chain defined maximum
	// gas limit, missing, 0, or improperly formatted, then the callbacks middleware will set it to the maximum gas
	// limit. In other words, `min(ctx.GasLimit, UserDefinedGasLimit())`.
	GetSourceUserDefinedGasLimit() uint64

	// GetDestUserDefinedGasLimit allows the sender of the packet to define the minimum amount of gas that the
	// relayer must set for the destination callback executions. If this value is greater than the chain defined
	// maximum gas limit, missing, 0, or improperly formatted, then the callbacks middleware will set it to the
	// maximum gas limit. In other words, `min(ctx.GasLimit, UserDefinedGasLimit())`.
	GetDestUserDefinedGasLimit() uint64

	// GetPacketSender returns the sender address of the packet.
	// If the packet sender is unknown, or undefined, an empty string should be returned.
	GetPacketSender(srcPortID, srcChannelID string) string
}
