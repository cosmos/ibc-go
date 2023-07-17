package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
)

// ContractKeeper defines the entry points to a smart contract that must be exposed by the VM module
type ContractKeeper interface {
	// IBCSendPacketCallback is called in the source chain when a PacketSend is executed. The
	// packetSenderAddress is determined by the underlying module, and may be empty if the sender is
	// unknown or undefined. The contract is expected to handle the callback within the user defined
	// gas limit, and handle any errors, or panics gracefully. The state will be reverted by the
	// middleware if an error is returned.
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
	// The state will be reverted by the middleware if an error is returned.
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
	// The state will be reverted by the middleware if an error is returned.
	IBCOnTimeoutPacketCallback(
		ctx sdk.Context,
		packet channeltypes.Packet,
		relayer sdk.AccAddress,
		contractAddress,
		packetSenderAddress string,
	) error
	// IBCWriteAcknowledgementCallback is called in the destination chain when a packet acknowledgement is written.
	// The packetReceiverAddress is determined by the underlying module, and may be empty if the sender
	// is unknown or undefined. The contract is expected to handle the callback within the user defined
	// gas limit, and handle any errors, out of gas, or panics gracefully.
	// The state will be reverted by the middleware if an error is returned.
	IBCWriteAcknowledgementCallback(
		ctx sdk.Context,
		packet ibcexported.PacketI,
		ack ibcexported.Acknowledgement,
		contractAddress,
		packetReceiverAddress string,
	) error
}
