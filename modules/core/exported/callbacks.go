package exported

type CallbackPacketData interface {
	// may return the empty string
	GetSrcCallbackAddress() string

	// may return the empty string
	GetDestCallbackAddress() string

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
