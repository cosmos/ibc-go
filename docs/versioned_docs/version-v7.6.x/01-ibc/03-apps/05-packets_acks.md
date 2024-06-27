---
title: Define packets and acks
sidebar_label: Define packets and acks
sidebar_position: 5
slug: /ibc/apps/packets_acks
---


# Define packets and acks

:::note Synopsis
Learn how to define custom packet and acknowledgement structs and how to encode and decode them. 
:::

:::note

## Pre-requisites Readings

- [IBC Overview](../01-overview.md))
- [IBC default integration](../02-integration.md)

:::

## Custom packets

Modules connected by a channel must agree on what application data they are sending over the
channel, as well as how they will encode/decode it. This process is not specified by IBC as it is up
to each application module to determine how to implement this agreement. However, for most
applications this will happen as a version negotiation during the channel handshake. While more
complex version negotiation is possible to implement inside the channel opening handshake, a very
simple version negotiation is implemented in the [ibc-transfer module](https://github.com/cosmos/ibc-go/tree/main/modules/apps/transfer/module.go).

Thus, a module must define its custom packet data structure, along with a well-defined way to
encode and decode it to and from `[]byte`.

```go
// Custom packet data defined in application module
type CustomPacketData struct {
    // Custom fields ...
}

EncodePacketData(packetData CustomPacketData) []byte {
    // encode packetData to bytes
}

DecodePacketData(encoded []byte) (CustomPacketData) {
    // decode from bytes to packet data
}
```

> Note that the `CustomPacketData` struct is defined in the proto definition and then compiled by the protobuf compiler.

Then a module must encode its packet data before sending it through IBC.

```go
// retrieve the dynamic capability for this channel
channelCap := scopedKeeper.GetCapability(ctx, channelCapName)
// Sending custom application packet data
data := EncodePacketData(customPacketData)
// Send packet to IBC, authenticating with channelCap
sequence, err := IBCChannelKeeper.SendPacket(
    ctx, 
    channelCap, 
    sourcePort, 
    sourceChannel, 
    timeoutHeight, 
    timeoutTimestamp, 
    data,
)
```

A module receiving a packet must decode the `PacketData` into a structure it expects so that it can
act on it.

```go
// Receiving custom application packet data (in OnRecvPacket)
packetData := DecodePacketData(packet.Data)
// handle received custom packet data
```

### Optional interfaces

The following interfaces are optional and MAY be implemented by a custom packet type.
They allow middlewares such as callbacks to access information stored within the packet data. 

#### PacketData interface

The `PacketData` interface is defined as follows:

```go
// PacketData defines an optional interface which an application's packet data structure may implement.
type PacketData interface {
	// GetPacketSender returns the sender address of the packet data.
	// If the packet sender is unknown or undefined, an empty string should be returned.
	GetPacketSender(sourcePortID string) string
}
```

The implementation of `GetPacketSender` should return the sender of the packet data. 
If the packet sender is unknown or undefined, an empty string should be returned.

This interface is intended to give IBC middlewares access to the packet sender of a packet data type. 

#### PacketDataProvider interface

The `PacketDataProvider` interface is defined as follows:

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

The implementation of `GetCustomPacketData` should return packet data held on behalf of another application (if present and supported). 
If this functionality is not supported, it should return nil. Otherwise it should return the packet data associated with the provided key. 

This interface gives IBC applications access to the packet data information embedded into the base packet data type. 
Within transfer and interchain accounts, the embedded packet data is stored within the Memo field. 

Once all IBC applications within an IBC stack are capable of creating/maintaining their own packet data type's, this interface function will be deprecated and removed. 

## Acknowledgements

Modules may commit an acknowledgement upon receiving and processing a packet in the case of synchronous packet processing.
In the case where a packet is processed at some later point after the packet has been received (asynchronous execution), the acknowledgement
will be written once the packet has been processed by the application which may be well after the packet receipt.

NOTE: Most blockchain modules will want to use the synchronous execution model in which the module processes and writes the acknowledgement
for a packet as soon as it has been received from the IBC module.

This acknowledgement can then be relayed back to the original sender chain, which can take action
depending on the contents of the acknowledgement.

Just as packet data was opaque to IBC, acknowledgements are similarly opaque. Modules must pass and
receive acknowledegments with the IBC modules as byte strings.

Thus, modules must agree on how to encode/decode acknowledgements. The process of creating an
acknowledgement struct along with encoding and decoding it, is very similar to the packet data
example above. [ICS 04](https://github.com/cosmos/ibc/blob/master/spec/core/ics-004-channel-and-packet-semantics#acknowledgement-envelope)
specifies a recommended format for acknowledgements. This acknowledgement type can be imported from
[channel types](https://github.com/cosmos/ibc-go/tree/main/modules/core/04-channel/types).

While modules may choose arbitrary acknowledgement structs, a default acknowledgement types is provided by IBC [here](https://github.com/cosmos/ibc-go/blob/main/proto/ibc/core/channel/v1/channel.proto):

```protobuf
// Acknowledgement is the recommended acknowledgement format to be used by
// app-specific protocols.
// NOTE: The field numbers 21 and 22 were explicitly chosen to avoid accidental
// conflicts with other protobuf message formats used for acknowledgements.
// The first byte of any message with this format will be the non-ASCII values
// `0xaa` (result) or `0xb2` (error). Implemented as defined by ICS:
// https://github.com/cosmos/ibc/tree/master/spec/core/ics-004-channel-and-packet-semantics#acknowledgement-envelope
message Acknowledgement {
  // response contains either a result or an error and must be non-empty
  oneof response {
    bytes  result = 21;
    string error  = 22;
  }
}
```
