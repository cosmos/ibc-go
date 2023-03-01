<!--
order: 1
-->

# `09-localhost`

## Overview

Learn about the 09-localhost light client module. {synopsis}

The 09-localhost light client module implements a localhost loopback client with the ability to send and receive IBC packets to and from the same state machine.

### Context

In a multichain environment, application developers will be used to developing cross-chain applications through IBC. From their point of view, whether or not they are interacting with multiple modules on the same chain or on different chains should not matter. The localhost client module enables a unified interface to interact with different applications on a single chain, using the familiar IBC application layer semantics.

### Implementation

There exists a [single sentinel `ClientState`](./client-state.md) instance with the client identifier `09-localhost`.

To supplement this, a [sentinel `ConnectionEnd` is stored in core IBC](./connection.md) state with the connection identifier `connection-localhost`. This enables IBC applications to create channels directly on top of the sentinel connection which leverage the 09-localhost loopback functionality.

[State verification](./state-verification.md) for channel state in handshakes or processing packets is reduced in complexity, the `09-localhost` client can simply compare bytes stored under the standardized key paths.

### Localhost vs *regular* client

The localhost client aims to provide a unified approach to interacting with applications on a single chain, as the IBC application layer provides for cross-chain interactions. To achieve this unified interface though, there are a number of differences under the hood compared to a 'regular' IBC client (excluding `06-solomachine` and `09-localhost` itself).

The table below lists some important differences:

|  | Regular client | Localhost |
| - | -------------- | --------- |
| Number of clients | many instances of a client *type* corresponding to different counterparties | 1 single sentinel client with the client identifier `09-localhost`|
| Client creation | Relayer (permissionless) | the 09-localhost `ClientState` is instantiated in the `InitGenesis` handler of the 02-client submodule in core IBC |
| Number of connection(s) | many connections, 1 (or more) per client | 1 single sentinel connection with the connection identifier `connection-localhost` |
| Connection creation | connection handshake, provided underlying client | the `ConnectionEnd` is created and set in store via the `InitGenesis` handler of the 03-connection submodule in core IBC |
| Counterparty | underlying client, representing another chain | client with identifier `09-localhost` in same chain |
| Channel creation | channel handshake, provided underlying connection | channel handshake, **out of the box** |
| Client updates | not automatic: relayer submitting `MsgUpdateClient` |automatic: latest height is updated periodically through the ABCI [`BeginBlock`](https://docs.cosmos.network/v0.47/building-modules/beginblock-endblock) interface of the 02-client submodule in core IBC |
| `VerifyMembership` and `VerifyNonMembership` | Performs proof verification using consensus state roots | Performs state verification using key-value lookups in the core IBC store |
