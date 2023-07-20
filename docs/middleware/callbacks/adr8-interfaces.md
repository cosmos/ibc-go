<!--
order: 1
-->

# Relevant Interfaces

To support [ADR8 defined packet callbacks](https://github.com/cosmos/ibc-go/tree/main/docs/architecture/adr-008-app-caller-cbs), an application module, middleware that wraps an ibc application, or smart contract [`PacketActor`](https://github.com/cosmos/ibc-go/blob/main/docs/architecture/adr-008-app-caller-cbs/adr-008-app-caller-cbs.md#decision) must implement two types of interfaces.

The`CallbackPacketData` interface is used to get the necessary data for executing a callback associated with a specific packet data type. The methods defined in this interface should be able to parse an optional [memo](https://github.com/cosmos/ibc-go/pull/3287/files#diff-789b0526436120518abd3a52d5f6118f3453d1016ac15bf58affbabd46bbac27R66) string associated with each packet, and return the data specified in the interface below.

In `ibc-go`, this interface has now been implemented for the ICS20 transfer, ICS27 Interchain Accounts packet types, and allows callback middleware to retrieve the desired callback addresses as well as callback gas limits for these packet types on the source and destination chains.

```go
type CallbackPacketData interface {
 // GetSourceCallbackAddress should return the callback address of a packet data on the source chain.
 // This may or may not be the sender of the packet. If no source callback address exists for the packet,
 // an empty string may be returned.
 GetSourceCallbackAddress() string

 // GetDestCallbackAddress should return the callback address of a packet data on the destination chain.
 // This may or may not be the receiver of the packet. If no dest callback address exists for the packet,
 // an empty string may be returned.
 GetDestCallbackAddress() string

 // GetSourceUserDefinedGasLimit allows the sender of the packet to define inside the packet data
 // a gas limit for how much the ADR-8 source callbacks can consume. If defined, this will be passed
 // in as the gas limit so that the callback is guaranteed to complete within a specific limit.
 // 
 // In the case of OnAcknowledgePacket and OnTimeoutPacket, a gas-overflow will reject state changes made during callback but still
 // commit the transaction. This ensures the packet lifecycle can always complete.
 //
 // If the packet data returns 0, the remaining gas limit will be passed in (modulo any chain-defined limit)
 // Otherwise, we will set the gas limit passed into the callback to the `min(ctx.GasLimit, UserDefinedGasLimit())`
 GetSourceUserDefinedGasLimit() uint64

// GetDestUserDefinedGasLimit allows the sender of the packet to define inside the packet data
 // a gas limit for how much the ADR-8 destination callbacks can consume. If defined, this will be passed
 // in as the gas limit so that the callback is guaranteed to complete within a specific limit.
 //
 // In the case of OnRecvPacket, a gas-overflow will just fail the transaction allowing it to timeout on the sender side.
 //
 // If the packet data returns 0, the remaining gas limit will be passed in (modulo any chain-defined limit)
 // Otherwise, we will set the gas limit passed into the callback to the `min(ctx.GasLimit, UserDefinedGasLimit())`
 GetDestUserDefinedGasLimit() uint64
    }
```

The second `ContractKeeper` interface allows for the execution of callbacks which are specific to each `PacketActor`, and defines the callback logic that will be executed at each step of the packet lifecycle.

```go
type ContractKeeper interface {
 // IBCSendPacketCallback is called in the source chain when a PacketSend is executed. The
 // packetSenderAddress is determined by the underlying module, and may be empty if the sender is
 // unknown or undefined.
 //
 // The contract is expected to handle the callback within the user defined
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
```

To allow for current `ibc-go` applications to handle ADR8 callbacks, a `callbacks` middleware has been implemented which can wrap any existing `ibc-go` application. This middleware expects the underlying application to have implemented the `CallbackPacketData` interface described above as well as an implementation of the `ContractKeeper` interface. It grabs the callback address and gas limit information from this interface, and then executes the relevant callback specified by the `ContractKeeper` at each step of the packet lifecycle.

For more detail on this middleware, please check out [the middleware specific documentation](../../ibc/middleware/callbacks/callbacks.md).

# Optional Interface (WIP)
*Note that this interface is still subject to change*

`PacketInfoProvider` defines an optional interface which allows a middleware to request the packet data to be unmarshaled by the base application.

```go
type PacketInfoProvider interface {
 // UnmarshalPacketData unmarshals the packet data into a concrete type
 UnmarshalPacketData([]byte) (interface{}, error)

 // GetPacketSender returns the sender address of the packet.
 // If the packet sender is unknown, or undefined, an empty string should be returned.
 GetPacketSender(packet exported.PacketI) string

 // GetPacketReceiver returns the receiver address of the packet.
 // If the packet receiver is unknown, or undefined, an empty string should be returned.
 GetPacketReceiver(packet exported.PacketI) string
}
```
