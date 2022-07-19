<!--
order: 1
-->

# IBC Applications

Learn how to build custom IBC application modules that enable packets to be sent to and received from other IBC-enabled chains. {synopsis}

This document serves as a guide for developers who want to write their own Inter-blockchain Communication Protocol (IBC) applications for custom use cases.

Due to the modular design of the IBC protocol, IBC application developers do not need to concern themselves with the low-level details of clients, connections, and proof verification. Nevertheless, an overview of these low-level concepts can be found in [the Overview section](../overview.md).
The document goes into detail on the abstraction layer most relevant for application developers (channels and ports), and describes how to define your own custom packets, `IBCModule` callbacks and more to make an application module IBC ready.

**To have your module interact over IBC you must:**

- implement the `IBCModule` interface, i.e.:
  - channel (opening) handshake callbacks
  - channel closing handshake callbacks
  - packet callbacks
- bind to a port(s)
- add keeper methods
- define your own packet data and acknowledgement structs as well as how to encode/decode them
- add a route to the IBC router

The following sections provide a more detailed explanation of how to write an IBC application
module correctly corresponding to the listed steps.

## Pre-requisites Readings

- [IBC Overview](../overview.md)) {prereq}
- [IBC default integration](../integration.md) {prereq}

## Working example

For a real working example of an IBC application, you can look through the `ibc-transfer` module
which implements everything discussed in this section.

Here are the useful parts of the module to look at:

[Binding to transfer
port](https://github.com/cosmos/ibc-go/blob/main/modules/apps/transfer/keeper/genesis.go)

[Sending transfer
packets](https://github.com/cosmos/ibc-go/blob/main/modules/apps/transfer/keeper/relay.go)

[Implementing IBC
callbacks](https://github.com/cosmos/ibc-go/blob/main/modules/apps/transfer/ibc_module.go)

## Next {hide}

Learn about [building modules](https://github.com/cosmos/cosmos-sdk/blob/master/docs/building-modules/intro.md) {hide}
