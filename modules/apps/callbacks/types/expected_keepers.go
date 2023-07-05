package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// ContractKeeper defines the entry points to a smart contract that must be exposed by the VM module
type ContractKeeper interface {
	// IBCAcknowledgementPacketCallback is called in the source chain when a packet acknowledgement
	// is received. The contract is expected to handle the callback within the user defined
	// gas limit, and handle any errors, or panics gracefully.
	// The user may also pass a custom message to the contract. (May be nil)
	// The state will be reverted by the middleware if an error is returned.
	IBCAcknowledgementPacketCallback(
		ctx sdk.Context,
		packet channeltypes.Packet,
		customMsg []byte,
		ackResult channeltypes.Acknowledgement,
		relayer sdk.AccAddress,
		contractAddr string,
	) error
	// IBCPacketTimeoutCallback is called in the source chain when a packet is not received before
	// the timeout height. The contract is expected to handle the callback within the user defined
	// gas limit, and handle any error, out of gas, or panics gracefully.
	// The state will be reverted by the middleware if an error is returned.
	IBCPacketTimeoutCallback(
		ctx sdk.Context,
		packet channeltypes.Packet,
		relayer sdk.AccAddress,
		contractAddr string,
	) error
	// IBCReceivePacketCallback is called in the destination chain when a packet is received.
	// The contract is expected to handle the callback within the user defined gas limit, and
	// handle any errors, out of gas, or panics gracefully.
	// The user may also pass a custom message to the contract. (May be nil)
	// The state will be reverted by the middleware if an error is returned.
	IBCReceivePacketCallback(
		ctx sdk.Context,
		packet channeltypes.Packet,
		customMsg []byte,
		ackResult channeltypes.Acknowledgement,
		relayer sdk.AccAddress,
		contractAddr string,
	) error
}
