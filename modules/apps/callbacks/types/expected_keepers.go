package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
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
		chanCap *capabilitytypes.Capability,
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
	// IBCOnRecvPacketCallback is called in the destination chain when a packet is received.
	// The packetReceiverAddress is determined by the underlying module, and may be empty if the sender
	// is unknown or undefined. The contract is expected to handle the callback within the user defined
	// gas limit, and handle any errors, out of gas, or panics gracefully.
	// The state will be reverted by the middleware if an error is returned.
	IBCOnRecvPacketCallback(
		ctx sdk.Context,
		packet channeltypes.Packet,
		acknowledgement ibcexported.Acknowledgement,
		relayer sdk.AccAddress,
		contractAddress,
		packetReceiverAddress string,
	) error
}
