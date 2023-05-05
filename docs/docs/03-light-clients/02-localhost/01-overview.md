---
title: Overview
sidebar_label: Overview
sidebar_position: 1
slug: /ibc/light-clients/localhost/overview
---


# `09-localhost`

## Overview

:::note Synopsis
Learn about the 09-localhost light client module.
:::

The 09-localhost light client module implements a localhost loopback client with the ability to send and receive IBC packets to and from the same state machine.

### Context

In a multichain environment, application developers will be used to developing cross-chain applications through IBC. From their point of view, whether or not they are interacting with multiple modules on the same chain or on different chains should not matter. The localhost client module enables a unified interface to interact with different applications on a single chain, using the familiar IBC application layer semantics.

### Implementation

There exists a [single sentinel `ClientState`](03-client-state.md) instance with the client identifier `09-localhost`.

To supplement this, a [sentinel `ConnectionEnd` is stored in core IBC](04-connection.md) state with the connection identifier `connection-localhost`. This enables IBC applications to create channels directly on top of the sentinel connection which leverage the 09-localhost loopback functionality.

[State verification](05-state-verification.md) for channel state in handshakes or processing packets is reduced in complexity, the `09-localhost` client can simply compare bytes stored under the standardized key paths.

### Localhost vs *regular* client

The localhost client aims to provide a unified approach to interacting with applications on a single chain, as the IBC application layer provides for cross-chain interactions. To achieve this unified interface though, there are a number of differences under the hood compared to a 'regular' IBC client (excluding `06-solomachine` and `09-localhost` itself).

The table below lists some important differences:

|                                              | Regular client | Localhost |
| -------------------------------------------- | --------------------------------------------------------------------------- | --------- |
| Number of clients                            | Many instances of a client *type* corresponding to different counterparties | A single sentinel client with the client identifier `09-localhost`|
| Client creation                              | Relayer (permissionless) | `ClientState` is instantiated in the `InitGenesis` handler of the 02-client submodule in core IBC |
| Client updates                               | Relayer submits headers using `MsgUpdateClient` | Latest height is updated periodically through the ABCI [`BeginBlock`](https://docs.cosmos.network/v0.47/building-modules/beginblock-endblock) interface of the 02-client submodule in core IBC |
| Number of connections                        | Many connections, 1 (or more) per client | A single sentinel connection with the connection identifier `connection-localhost` |
| Connection creation                          | Connection handshake, provided underlying client | Sentinel `ConnectionEnd` is created and set in store in the `InitGenesis` handler of the 03-connection submodule in core IBC |
| Counterparty                                 | Underlying client, representing another chain | Client with identifier `09-localhost` in same chain |
| `VerifyMembership` and `VerifyNonMembership` | Performs proof verification using consensus state roots | Performs state verification using key-value lookups in the core IBC store |
| Integration | Expected to register codec types using the `AppModuleBasic` interface | Registers codec types within the core IBC module |
