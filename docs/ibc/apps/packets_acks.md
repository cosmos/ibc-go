<!--
order: 5
-->

# Define packets and acks

Learn how to define custom packet and acknowledgement structs and how to encode and decode them. {synopsis}

## Pre-requisites Readings

- [IBC Overview](../overview.md)) {prereq}
- [IBC default integration](../integration.md) {prereq}

## Custom packets

Modules connected by a channel must agree on what application data they are sending over the
channel, as well as how they will encode/decode it. This process is not specified by IBC as it is up
to each application module to determine how to implement this agreement. However, for most
applications this will happen as a version negotiation during the channel handshake. While more
complex version negotiation is possible to implement inside the channel opening handshake, a very
simple version negotation is implemented in the [ibc-transfer module](https://github.com/cosmos/ibc-go/tree/main/modules/apps/transfer/module.go).

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
// Sending custom application packet data
data := EncodePacketData(customPacketData)
packet.Data = data
IBCChannelKeeper.SendPacket(ctx, packet)
```

A module receiving a packet must decode the `PacketData` into a structure it expects so that it can
act on it.

```go
// Receiving custom application packet data (in OnRecvPacket)
packetData := DecodePacketData(packet.Data)
// handle received custom packet data
```

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
