package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// ContractKeeper defines the entry points exposed to the VM module which invokes a smart contract
type ContractKeeper interface {
	// IBCSendPacketCallback is called in the source chain when a PacketSend is executed. The
	// packetSenderAddress is determined by the underlying module, and may be empty if the sender is
	// unknown or undefined. The contract is expected to handle the callback within the user defined
	// gas limit, and handle any errors, or panics gracefully.
	// This entry point is called with a cached context. If an error is returned, then the changes in
	// this context will not be persisted, and the error will be propagated to the underlying IBC
	// application, resulting in a packet send failure.
	//
	// Implementations are provided with the packetSenderAddress and MAY choose to use this to perform
	// validation on the origin of a given packet. It is recommended to perform the same validation
	// on all source chain callbacks (SendPacket, AcknowledgementPacket, TimeoutPacket). This
	// defensively guards against exploits due to incorrectly wired SendPacket ordering in IBC stacks.
	IBCSendPacketCallback(
		cachedCtx sdk.Context,
		sourcePort string,
		sourceChannel string,
		timeoutHeight clienttypes.Height,
		timeoutTimestamp uint64,
		packetData []byte,
		contractAddress,
		packetSenderAddress string,
	) error
	// IBCOnAcknowledgementPacketCallback is called in the source chain when a packet acknowledgement
	// is received. The packetSenderAddress is determined by the underlying module, and may be empty if
	// the sender is unknown or undefined. The contract is expected to handle the callback within the
	// user defined gas limit, and handle any errors, or panics gracefully.
	// This entry point is called with a cached context. If an error is returned, then the changes in
	// this context will not be persisted, but the packet lifecycle will not be blocked.
	//
	// Implementations are provided with the packetSenderAddress and MAY choose to use this to perform
	// validation on the origin of a given packet. It is recommended to perform the same validation
	// on all source chain callbacks (SendPacket, AcknowledgementPacket, TimeoutPacket). This
	// defensively guards against exploits due to incorrectly wired SendPacket ordering in IBC stacks.
	IBCOnAcknowledgementPacketCallback(
		cachedCtx sdk.Context,
		packet channeltypes.Packet,
		acknowledgement []byte,
		relayer sdk.AccAddress,
		contractAddress,
		packetSenderAddress string,
	) error
	// IBCOnTimeoutPacketCallback is called in the source chain when a packet is not received before
	// the timeout height. The packetSenderAddress is determined by the underlying module, and may be
	// empty if the sender is unknown or undefined. The contract is expected to handle the callback
	// within the user defined gas limit, and handle any error, out of gas, or panics gracefully.
	// This entry point is called with a cached context. If an error is returned, then the changes in
	// this context will not be persisted, but the packet lifecycle will not be blocked.
	//
	// Implementations are provided with the packetSenderAddress and MAY choose to use this to perform
	// validation on the origin of a given packet. It is recommended to perform the same validation
	// on all source chain callbacks (SendPacket, AcknowledgementPacket, TimeoutPacket). This
	// defensively guards against exploits due to incorrectly wired SendPacket ordering in IBC stacks.
	IBCOnTimeoutPacketCallback(
		cachedCtx sdk.Context,
		packet channeltypes.Packet,
		relayer sdk.AccAddress,
		contractAddress,
		packetSenderAddress string,
	) error
	// IBCReceivePacketCallback is called in the destination chain when a packet acknowledgement is written.
	// The contract is expected to handle the callback within the user defined gas limit, and handle any errors,
	// out of gas, or panics gracefully.
	// This entry point is called with a cached context. If an error is returned, then the changes in
	// this context will not be persisted, but the packet lifecycle will not be blocked.
	IBCReceivePacketCallback(
		cachedCtx sdk.Context,
		packet ibcexported.PacketI,
		ack ibcexported.Acknowledgement,
		contractAddress string,
	) error
}
