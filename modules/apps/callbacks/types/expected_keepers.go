package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// ContractKeeper defines the entry points exposed to the VM module which invokes a smart contract
type ContractKeeper interface {
	// IBCSendPacketCallback is called in the source chain when a PacketSend is executed. The
	// packetSenderAddress is determined by the underlying module, and may be empty if the sender is
	// unknown or undefined. The contract is expected to handle the callback within the user defined
	// gas limit, and handle any errors, or panics gracefully.
	// If an error is returned, the transaction will be reverted by the callbacks middleware, and the
	// packet will not be sent.
	//
	// Implementations are provided with the packetSenderAddress and MAY choose to use this to perform
	// validation on the origin of a given packet. It is recommended to perform the same validation
	// on all source chain callbacks (SendPacket, AcknowledgementPacket, TimeoutPacket). This
	// defensively guards against exploits due to incorrectly wired SendPacket ordering in IBC stacks.
	IBCSendPacketCallback(
		ctx sdk.Context,
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
	// If an error is returned, state will be reverted by the callbacks middleware.
	//
	// Implementations are provided with the packetSenderAddress and MAY choose to use this to perform
	// validation on the origin of a given packet. It is recommended to perform the same validation
	// on all source chain callbacks (SendPacket, AcknowledgementPacket, TimeoutPacket). This
	// defensively guards against exploits due to incorrectly wired SendPacket ordering in IBC stacks.
	IBCOnAcknowledgementPacketCallback(
		ctx sdk.Context,
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
	// If an error is returned, state will be reverted by the callbacks middleware.
	//
	// Implementations are provided with the packetSenderAddress and MAY choose to use this to perform
	// validation on the origin of a given packet. It is recommended to perform the same validation
	// on all source chain callbacks (SendPacket, AcknowledgementPacket, TimeoutPacket). This
	// defensively guards against exploits due to incorrectly wired SendPacket ordering in IBC stacks.
	IBCOnTimeoutPacketCallback(
		ctx sdk.Context,
		packet channeltypes.Packet,
		relayer sdk.AccAddress,
		contractAddress,
		packetSenderAddress string,
	) error
	// IBCReceivePacketCallback is called in the destination chain when a packet acknowledgement is written.
	// The contract is expected to handle the callback within the user defined gas limit, and handle any errors,
	// out of gas, or panics gracefully.
	// If an error is returned, state will be reverted by the callbacks middleware.
	IBCReceivePacketCallback(
		ctx sdk.Context,
		packet ibcexported.PacketI,
		ack ibcexported.Acknowledgement,
		contractAddress string,
	) error
}
