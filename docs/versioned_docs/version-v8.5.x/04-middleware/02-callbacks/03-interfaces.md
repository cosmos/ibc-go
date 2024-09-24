---
title: Interfaces
sidebar_label: Interfaces
sidebar_position: 3
slug: /middleware/callbacks/interfaces
---

# Interfaces

The callbacks middleware requires certain interfaces to be implemented by the underlying IBC applications and the secondary application. If you're simply wiring up the callbacks middleware to an existing IBC application stack and a secondary application such as `icacontroller` and `x/wasm`, you can skip this section.

## Interfaces for developing the Underlying IBC Application

### `PacketDataUnmarshaler`

```go
// PacketDataUnmarshaler defines an optional interface which allows a middleware to
// request the packet data to be unmarshaled by the base application.
type PacketDataUnmarshaler interface {
  // UnmarshalPacketData unmarshals the packet data into a concrete type
  UnmarshalPacketData([]byte) (interface{}, error)
}
```

The callbacks middleware **requires** the underlying ibc application to implement the [`PacketDataUnmarshaler`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/core/05-port/types/module.go#L142-L147) interface so that it can unmarshal the packet data bytes into the appropriate packet data type. This allows usage of interface functions implemented by the packet data type. The packet data type is expected to implement the `PacketDataProvider` interface (see section below), which is used to parse the callback data that is currently stored in the packet memo field for `transfer` and `ica` packets as a JSON string. See its implementation in the [`transfer`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/apps/transfer/ibc_module.go#L303-L313) and [`icacontroller`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/apps/27-interchain-accounts/controller/ibc_middleware.go#L258-L268) modules for reference.

If the underlying application is a middleware itself, then it can implement this interface by simply passing the function call to its underlying application. See its implementation in the [`fee middleware`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/apps/29-fee/ibc_middleware.go#L368-L378) for reference.

### `PacketDataProvider`

```go
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
```

The callbacks middleware also **requires** the underlying ibc application's packet data type to implement the [`PacketDataProvider`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/core/exported/packet.go#L43-L52) interface. This interface is used to retrieve the callback data from the packet data (using the memo field in the case of `transfer` and `ica`). For example, see its implementation in the [`transfer`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/apps/transfer/types/packet.go#L85-L105) module.

Since middlewares do not have packet types, they do not need to implement this interface.

### `PacketData`

```go
// PacketData defines an optional interface which an application's packet data structure may implement.
type PacketData interface {
  // GetPacketSender returns the sender address of the packet data.
  // If the packet sender is unknown or undefined, an empty string should be returned.
  GetPacketSender(sourcePortID string) string
}
```

[`PacketData`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/core/exported/packet.go#L36-L41) is an optional interface that can be implemented by the underlying ibc application's packet data type. It is used to retrieve the packet sender address from the packet data. The callbacks middleware uses this interface to retrieve the packet sender address and pass it to the callback function during a source callback. If this interface is not implemented, then the callbacks middleware passes and empty string as the sender address. For example, see its implementation in the [`transfer`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/apps/transfer/types/packet.go#L74-L83) and [`ica`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/apps/27-interchain-accounts/types/packet.go#L78-L92) module.

This interface was added so that secondary applications can retrieve the packet sender address to perform custom authorization logic if needed.

Since middlewares do not have packet types, they do not need to implement this interface.

## Interfaces for developing the Secondary Application

### `ContractKeeper`

The callbacks middleware requires the secondary application to implement the [`ContractKeeper`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/apps/callbacks/types/expected_keepers.go#L11-L83) interface. The contract keeper will be invoked at each step of the packet lifecycle. When a packet is sent, if callback information is provided, the contract keeper will be invoked via the `IBCSendPacketCallback`. This allows the contract keeper to prevent packet sends when callback information is provided, for example if the sender is unauthorized to perform callbacks on the given information. If the packet send is successful, the contract keeper on the destination (if present) will be invoked when a packet has been received and the acknowledgement is written, this will occur via `IBCReceivePacketCallback`. At the end of the packet lifecycle, when processing acknowledgements or timeouts, the source contract keeper will be invoked either via `IBCOnAcknowledgementPacket` or `IBCOnTimeoutPacket`. Once a packet has been sent, each step of the packet lifecycle can be processed given that a relayer sets the gas limit to be more than or equal to the required `CommitGasLimit`. State changes performed in the callback will only be committed upon successful execution.

```go
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
```

These are the callback entry points exposed to the secondary application. The secondary application is expected to execute its custom logic within these entry points. The callbacks middleware will handle the execution of these callbacks and revert the state if needed.

:::tip
Note that the source callback entry points are provided with the `packetSenderAddress` and MAY choose to use this to perform validation on the origin of a given packet. It is recommended to perform the same validation on all source chain callbacks (SendPacket, AcknowledgePacket, TimeoutPacket). This defensively guards against exploits due to incorrectly wired SendPacket ordering in IBC stacks.
:::
