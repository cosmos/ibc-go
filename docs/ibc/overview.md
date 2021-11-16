<!--
order: false
parent: 
  order: 1
-->

# Overview

Learn what IBC is, its components and use cases. {synopsis}

## What is the Interblockchain Communication Protocol (IBC)?

This document serves as a guide for developers who want to write their own Inter-Blockchain
Communication protocol (IBC) applications for custom use cases.

Due to the modular design of the IBC protocol, IBC
application developers do not need to be concerned with the low-level details of clients,
connections, and proof verification. 

This brief explanation of the lower levels of the
stack is provided to give application developers a high-level understanding of the IBC
protocol. Details on the abstraction layer are most relevant for application
developers (channels and ports) and describe how to define custom packets and
`IBCModule` callbacks.

To have your module interact over IBC you must: 

- Bind to a port or ports
- Define your own packet data
- Define optional acknowledgement structs
- Know how to encode and decode the packet data
- Implement the `IBCModule` interface

Read on for a detailed explanation of how to write an IBC application
module correctly.

## Components Overview

### [Clients](https://github.com/cosmos/ibc-go/blob/main/modules/core/02-client)

IBC clients are light clients that are identified by a unique client-id. IBC clients track the consensus states of
other blockchains, along with the proof spec necessary to properly verify proofs against the
client's consensus state. A client can be associated with any number of connections to the counterparty
chain. 

The supported IBC clients are:

* [Solo Machine light client](https://github.com/cosmos/ibc-go/blob/main/modules/light-clients/06-solomachine): Devices such as phones, browsers, or laptops.
* [Tendermint light client](https://github.com/cosmos/ibc-go/blob/main/modules/light-clients/07-tendermint): The default for Cosmos SDK-based chains.
* [Localhost (loopback) client](https://github.com/cosmos/ibc-go/blob/main/modules/light-clients/09-localhost): Useful for
testing, simulation, and relaying packets to modules on the same application.

### [Connections](https://github.com/cosmos/ibc-go/blob/main/modules/core/03-connection)

Connections encapsulate two `ConnectionEnd` objects on two separate blockchains. Each
`ConnectionEnd` is associated with a client of the other blockchain (for example, the counterparty blockchain).
The connection handshake is responsible for verifying that the light clients on each chain are
correct for their respective counterparties. Connections, once established, are responsible for
facilitation all cross-chain verification of IBC state. A connection can be associated with any
number of channels.

### [Proofs](https://github.com/cosmos/ibc-go/blob/main/modules/core/23-commitment) and [Paths](https://github.com/cosmos/ibc-go/blob/main/modules/core/24-host)
  
In IBC, blockchains do not directly pass messages to each other over the network. Instead, to
communicate, a blockchain commits some state to a specifically defined path that is reserved for a
specific message type and a specific counterparty. For example, for storing a specific connectionEnd as part
of a handshake, or a packet intended to be relayed to a module on the counterparty chain. A relayer
process monitors for updates to these paths and relays messages by submitting the data stored
under the path along with a proof to the counterparty chain. 

- The paths that all IBC implementations must use for committing IBC messages is defined in
[ICS-24 Host State Machine Requirements](https://github.com/cosmos/ics/tree/master/spec/core/ics-024-host-requirements). 
- The proof format that all implementations must be able to produce and verify is defined in [ICS-23 Proofs](https://github.com/confio/ics23) implementation.

### [Capabilities](https://github.com/cosmos/cosmos-sdk/blob/master/docs/core/ocap.md)

IBC is intended to work in execution environements where modules do not necessarily trust each
other. Thus, IBC must authenticate module actions on ports and channels so that only modules with the
appropriate permissions can use them. This module action authentication is accomplished using a [dynamic
capability store](https://github.com/cosmos/cosmos-sdk/blob/master/docs/architecture/adr-003-dynamic-capability-store.md). Upon binding to a port or
creating a channel for a module, IBC returns a dynamic capability that the module must claim in
order to use that port or channel. This dynamic capability store prevents other modules from using that port or channel since
they do not own the appropriate capability.

While this background information is useful, IBC modules do not need to interact at all with
these lower-level abstractions. The relevant abstraction layer for IBC application developers is
that of channels and ports. IBC applications must be written as self-contained **modules**. 

A module on one blockchain can thus communicate with other modules on other blockchains by sending,
receiving, and acknowledging packets through channels that are uniquely identified by the
`(channelID, portID)` tuple. 

A useful analogy is to consider IBC modules as internet applications on
a computer. A channel can then be conceptualized as an IP connection, with the IBC portID being
analogous to a IP port and the IBC channelID being analogous to an IP address. Thus, a single
instance of an IBC module can communicate on the same port with any number of other modules and
IBC correctly routes all packets to the relevant module using the (channelID, portID tuple). An
IBC module can also communicate with another IBC module over multiple ports, with each
`(portID<->portID)` packet stream being sent on a different unique channel.

### [Ports](https://github.com/cosmos/ibc-go/blob/main/modules/core/05-port)

An IBC module can bind to any number of ports. Each port must be identified by a unique `portID`.
Since IBC is designed to be secure with mutually-distrusted modules operating on the same ledger,
binding a port returns a dynamic object capability. In order to take action on a particular port
(for example, an open a channel with its portID), a module must provide the dynamic object capability to the IBC
handler. This requirement prevents a malicious module from opening channels with ports it does not own. Thus,
IBC modules are responsible for claiming the capability that is returned on `BindPort`.

### [Channels](https://github.com/cosmos/ibc-go/blob/main/modules/core/04-channel)

An IBC channel can be established between two IBC ports. Currently, a port is exclusively owned by a
single module. IBC packets are sent over channels. Just as IP packets contain the destination IP
address and IP port as well as the source IP address and source IP port, IBC packets contain
the destination portID and channelID as well as the source portID and channelID. This packet structure enables IBC to
correctly route packets to the destination module while also allowing modules receiving packets to
know the sender module.

A channel can be `ORDERED`, in which case, packets from a sending module must be processed by the
receiving module in the order they were sent. Or a channel can be `UNORDERED`, in which case packets
from a sending module are processed in the order they arrive (might be a different order than they were sent).

Modules can choose which channels they wish to communicate over with, thus IBC expects modules to
implement callbacks that are called during the channel handshake. These callbacks can do custom
channel initialization logic. If any callback returns an error, the channel handshake fails. Thus, by
returning errors on callbacks, modules can programatically reject and accept channels.

The channel handshake is a 4-step handshake. Briefly, if a given chain A wants to open a channel with
chain B using an already established connection:

1. chain A sends a `ChanOpenInit` message to signal a channel initialization attempt with chain B.
2. chain B sends a `ChanOpenTry` message to try opening the channel on chain A.
3. chain A sends a `ChanOpenAck` message to mark its channel end status as open.
4. chain B sends a `ChanOpenConfirm` message to mark its channel end status as open.

If all this happens successfully, the channel is opened on both sides. At each step in the handshake, the module
associated with the `ChannelEnd` executes its callback for that step of the handshake. So
on `ChanOpenInit`, the module on chain A executes its callback `OnChanOpenInit`.

Just as ports came with dynamic capabilites, channel initialization returns a dynamic capability
that the module **must** claim so that they can pass in a capability to authenticate channel actions
like sending packets. The channel capability is passed into the callback on the first parts of the
handshake; either `OnChanOpenInit` on the initializing chain or `OnChanOpenTry` on the other chain.

### [Packets](https://github.com/cosmos/ibc-go/blob/main/modules/core/04-channel)

Modules communicate with each other by sending packets over IBC channels. As previously mentioned, all
IBC packets contain the destination `portID` and `channelID` along with the source `portID` and
`channelID`. This packet structure allows modules to know the sender module of a given packet. IBC packets 
contain a sequence to optionally enforce ordering. 

IBC packets also contain a `TimeoutTimestamp` and
`TimeoutHeight`, which when non-zero, determine the deadline before which the receiving module
must process a packet. If the timeout passes without the packet being successfully received, the
sending module can timeout the packet and take appropriate actions.

Modules send custom application data to each other inside the `Data []byte` field of the IBC packet.
Thus, packet data is completely opaque to IBC handlers. It is incumbent on a sender module to encode
their application-specific packet information into the `Data` field of packets, and the receiver
module to decode that `Data` back to the original application data.

### [Receipts and Timeouts](https://github.com/cosmos/ibc-go/blob/main/modules/core/04-channel)

Since IBC works over a distributed network and relies on potentially faulty relayers to relay messages between ledgers, 
IBC must handle the case where a packet does not get sent to its destination in a timely manner or at all. Thus, packets must 
specify a timeout height or timeout timestamp after which a packet can no longer be successfully received on the destination chain.

If the timeout does get reached, then a proof of packet timeout can be submitted to the original chain which can then perform 
application-specific logic to timeout the packet, perhaps by rolling back the packet send changes (refunding senders any locked funds, etc).

- In ORDERED channels, a timeout of a single packet in the channel causes the channel to close. If packet sequence `n` times out, 
then no packet at sequence `k > n` can be successfully received without violating the contract of ORDERED channels that packets are processed in the order that they are sent. Since ORDERED channels enforce this invariant, a proof that sequence `n` hasn't been received on the destination chain by the specified timeout of packet `n` is sufficient to timeout packet `n` and close the channel.

- In the UNORDERED case, packets can be received in any order. Thus, IBC writes a packet receipt for each sequence it has received in the UNORDERED channel. This receipt contains no information, it is simply a marker intended to signify that the UNORDERED channel has received a packet at the specified sequence. To timeout a packet on an UNORDERED channel, one must provide a proof that a packet receipt does not exist for the packet's sequence by the specified timeout. Of course, timing out a packet on an UNORDERED channel simply triggers the application specific timeout logic for that packet, and does not close the channel.

For this reason, most modules should use UNORDERED channels as they require less liveness guarantees to function effectively for users of that channel.

### [Acknowledgements](https://github.com/cosmos/ibc-go/blob/main/modules/core/04-channel)

Modules can also choose to write application-specific acknowledgements upon processing a packet. This acknowledgement can be done synchronously on `OnRecvPacket` if the module processes packets as soon as they are received from IBC module. Or acknowledgements can be done asynchronously if module processes packets at some later point after receiving the packet.

Regardless, this acknowledgement data is opaque to IBC much like the packet `Data` and is treated by IBC as a simple byte string `[]byte`. It is incumbent on receiver modules to encode their acknowledgemnet in such a way that the sender module can decode it correctly. This should be decided through version negotiation during the channel handshake.

The acknowledgement can encode whether the packet processing succeeded or failed, along with additional information that allows the sender module to take appropriate action.

After the acknowledgement has been written by the receiving chain, a relayer relays the acknowledgement back to the original sender module which  then executes application-specific acknowledgment logic using the contents of the acknowledgement. This can involve rolling back packet-send changes in the case of a failed acknowledgement (refunding senders).

After an acknowledgement is received successfully on the original sender the chain, the IBC module deletes the corresponding packet commitment as it is no longer needed.

## Further Readings and Specs

If you want to learn more about IBC, check the following specifications:

* [IBC specification overview](https://github.com/cosmos/ibc/blob/master/README.md)
* [IBC SDK specification](../../modules/core/spec/README.md)

## Next {hide}

Learn about how to [integrate](./integration.md) IBC to your application {hide}
