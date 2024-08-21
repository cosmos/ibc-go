---
title: Roadmap
sidebar_label: Roadmap
sidebar_position: 10
slug: /ibc/roadmap
---

# Roadmap ibc-go

*Latest update: August 21st, 2024*

This document endeavours to inform the wider IBC community about plans and priorities for work on ibc-go by the team at Interchain GmbH. It is intended to broadly inform all users of ibc-go, including developers and operators of IBC, relayer, chain and wallet applications.

This roadmap should be read as a high-level guide, rather than a commitment to schedules and deliverables. The degree of specificity is inversely proportional to the timeline. We will update this document periodically to reflect the status and plans.

## v9.0.0

Final release is scheduled for beginning of September 2024.

### 02-client router refactor

This refactor decouples [routing from encoding](https://github.com/cosmos/ibc-go/issues/5565). The light client's implementation of the `ClientState` interface is not used anymore to route requests to the right light client at the 02-client layer. Instead, a light client module is registered for every light client type (tendermint, solomachine, etc.) and 02-client routes the requests to the right light client module based on the client ID.

### ICS20 v2

The transfer application will be updated to add support for [transferring multiple tokens in the same packet](https://github.com/cosmos/ibc/pull/1020) and support for [atomically route tokens series of paths with a single packet](https://github.com/cosmos/ibc/pull/1090).

## v10.0.0

### 05-port refactor

The port router refactor aims to simplify how IBC application stacks are wired up and executed during application callbacks. The application stacks will be converted into an ordered list of callbacks for a given port ID. The purpose of this is to decouple applications from one another, while maintaining backwards compatibility with existing IBC usage. For more details, please check [#6985](https://github.com/cosmos/ibc-go/issues/6941).

Additionally, a new message type (`MsgSendPacket`) and its corresponding RPC handler will be added to the core IBC message server to allow users or applications to send packets. The sending if a packet will trigger the `OnSendPacket` callback. This new callback and the other existing callbacks for receiving, acknowledging and timing out will be invoked in order for the applications registered on a given port ID.

```proto
message MsgSendPacket {
  string                    source_port       = 1;
  string                    source_channel    = 2;
  ibc.core.client.v1.Height timeout_height    = 3 [(gogoproto.nullable) = false];
  uint64                    timeout_timestamp = 4;
  bytes                     data              = 5;
  string                    signer            = 6;
}
```

For more details, please check [#7012](https://github.com/cosmos/ibc-go/issues/7012).

### Multipacket execution

Building on top of the 05-port refactor, we will implement support for multiple packet data's to be delivered and executed within a single packet. This will allow applications wired together in a list of ordered applications to provide their own packet data. By introducing this feature, we are then able to specify in the packet the encoding and the version of each application's packet data, which will enable us to remove in the future channel handshakes and channel upgradability (as each packet data will indicate which application it is bound for, what version of the application it wishes to interact with and how it can be decoded).

Currently the `Packet` proto message look like the following:

```proto
message Packet {
  option (gogoproto.goproto_getters) = false;
  uint64 sequence = 1;
  string source_port = 2;
  string source_channel = 3;
  string destination_port = 4;
  string destination_channel = 5;
  bytes data = 6;
  ibc.core.client.v1.Height timeout_height = 7 [(gogoproto.nullable) = false];
  uint64 timeout_timestamp = 8;
}
```

Where the `data` field is the byte representation of the application packet data (e.g the bytes that result from serializing `FungibleTokenPacketData` in the case of transfer). This packet structure supports only a single application packet data. With multipacket support we will give each application in a ordered lists of applications the possibility to have their own packet data. Therefore, we will introduce a new packet structure that might look like this:

```proto
message PacketV2 {
  int64                     sequence            = 1;
  string                    source_port         = 2;
  string                    source_channel      = 3;
  string                    destination_port    = 4;
  string                    destination_channel = 5;
  repeated PacketData       data                = 6; // primary modification
  ibc.core.client.v1.Height timeout_height      = 7 [(gogoproto.nullable) = false];
  uint64                    timeout_timestamp   = 8;
}

message PacketData {
  string port_id  = 1; // or app_name
  string type     = 2; // or version
  string encoding = 3;
  bytes  value    = 4; // the serialized bytes of the application's packet data
}
```

The `PacketV2` proto message contains a list of items of type `PacketData`, which each contains the application port ID, the encoding used to serialized the packet data, packet data version (e.g. `ics20-1` or `ics20-2` in the case of transfer) and the serialized bytes of the packet data. Each packet data will then be delivered which can then be delivered to an individual IBC application. The `MsgSendPacket` proto message will change accordingly:

```diff
message MsgSendPacket {
  string                    source_port       = 1;
  string                    source_channel    = 2;
  ibc.core.client.v1.Height timeout_height    = 3 [(gogoproto.nullable) = false];
  uint64                    timeout_timestamp = 4;
- bytes                     data              = 5;
+ repeated PacketData       data              = 5;
  string                    signer            = 6;
}
```

In order to be able to send multiple packet data's we will introduce a new sentinel port ID to indicate a packet is sent using this new multi-packet data structure.

For more details, please check [#7008](https://github.com/cosmos/ibc-go/issues/7008).

### Eureka

IBC launched more than three years ago and the experience during this time has taught us that IBC, in its current design, has several drawbacks that complicate innovation on top of it and its expansion beyond the Cosmos ecosystem:

- Implementation of IBC on new platforms and environments is difficult and resource intensive.
- IBC's multiple layers of abstraction are burdensome, confusing, and unnecessary.
- IBC demands lots of state to be written and messages exchanged (for connection and channel handshakes) before the first user-level message can be send sent (e.g. before a transfer packet can be sent).
- Upgradability is difficult and slow, making innovation on IBC difficult and slow.

With the code-named project Eureka we want to address these problems by vastly simplifying the protocol, but without compromising or reducing security and functionality. At a high level, IBC Eureka:

- Removes connection and channel handshakes.
- Retains interoperability using a light client based security model.
- Retains existing packet semantics (send, receive, acknowledge, timeout).
- Uses a single light client and channel for all applications.

For more details, please check [#6985](https://github.com/cosmos/ibc-go/issues/6985). The initial Eureka design was proposed in [cosmos/ibc#1093](https://github.com/cosmos/ibc/pull/1093).

### Compatibility

In v10 there will be two implementations of the IBC protocal living side by side. We will call them classic (or v1) and eureka (or v2). The classic implementation remains completely backwards compatible with all previous versions of ibc-go (i.e. connection and channel handshakes, channel upgradability and packet processing will work as they do today), but the eureka implementation will be compatible only with other IBC Eureka implementations (e.g. [solidity-ibc-eureka](https://github.com/cosmos/solidity-ibc-eureka) for EVMs) and other chains running ibc-go v10.

Chains running ibc-go v10 have the choice then to communicate between them using either classic or eureka implementations. For example: given that chain A (with ibc-go v10) already had a transfer channel set up with chain B (also running ibc-go v10), if tokens need to be sent from chain A to chain B, then that can be done using the classic implementation or the eureka implementation. Most likely the indication of which implementation to use will be in a new `protocol_version` field in `MsgSendPacket`:

```proto
message MsgSendPacket {
  string                    source_port         = 1;
  string                    source_channel      = 2;
+ string                    destination_port    = 3;
+ string                    destination_channel = 4;
  ibc.core.client.v1.Height timeout_height      = 5 [(gogoproto.nullable) = false];
  uint64                    timeout_timestamp   = 6;
  repeated PacketData       data                = 7;
+ IBCVersion                protocol_version    = 8;
  string                    signer              = 9;
}

enum IBCVersion {
  IBC_VERSION_UNSPECIFIED = 0;
  // IBC version 1 implements the Classic protocol
  IBC_VERSION_1 = 1;
  // IBC version 2 implements the Eureka protocol
  IBC_VERSION_2 = 2;
}
```

For example, in the case of a token transfer, the existing source/destination port ID/channel ID can be used, so that even if the tokens are sent/received over the eureka implementation, they remain fungible with tokens previously sent using the classic implementation. The sending chain can specify either protocol v1 or v2 in the `protocol_version` field of `MsgSendPacket`.

Since IBC Eureka will allow communication directly over light clients (i.e. no need for channels), then for `source_channel` and `destination_channel` it will be also possible to use client IDs (instead of channel IDs) if two chains that did not have a channel set up previously want to communicate using the eureka implementation (so that they do not go through a channel handshake to be able to send packets to each other).

Chains on ibc-go v10 can still communicate with chains running earlier version of ibc-go, and that should be do using the classic implementation by specifiying the v1 protocol version in `MsgSendPacket`.

## v11.0.0

### Conditional packets

The multipacket support introduced in v10 will be improved to add atomic execution guarantees of all of the application's packet data. For more details, please check [#7003](https://github.com/cosmos/ibc-go/issues/7003).

### ICA v2

This new version of ICS27 will address many of [the pain points with the current design](https://github.com/cosmos/ibc-go/pull/6281), including multiplexing all communication between controller and host through a single channel (instead of each interchain account on the host being associated to a different channel, as it is now).

---

This roadmap is also available as a [project board](https://github.com/orgs/cosmos/projects/7/views/25).

For the latest expected release timelines, please check [here](https://github.com/cosmos/ibc-go/wiki/Release-timeline).

For the latest information on the progress of the work or the decisions made that might influence the roadmap, please follow the [Announcements](https://github.com/cosmos/ibc-go/discussions/categories/announcements) category in the Discussions tab of the repository.

---

**Note**: release version numbers may be subject to change.
