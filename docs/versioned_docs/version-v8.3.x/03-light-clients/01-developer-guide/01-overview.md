---
title: Overview
sidebar_label: Overview
sidebar_position: 1
slug: /ibc/light-clients/overview
---

# Overview

:::note Synopsis
Learn how to build IBC light client modules and fulfill the interfaces required to integrate with core IBC.
:::

:::note

## Pre-requisite readings

- [IBC Overview](../../01-ibc/01-overview.md)
- [IBC Transport, Authentication, and Ordering Layer - Clients](https://tutorials.cosmos.network/academy/3-ibc/4-clients.html)
- [ICS-002 Client Semantics](https://github.com/cosmos/ibc/tree/main/spec/core/ics-002-client-semantics)

:::

IBC uses light clients in order to provide trust-minimized interoperability between sovereign blockchains. Light clients operate under a strict set of rules which provide security guarantees for state updates and facilitate the ability to verify the state of a remote blockchain using merkle proofs.

The following aims to provide a high level IBC light client module developer guide. Access to IBC light clients are gated by the core IBC `MsgServer` which utilizes the abstractions set by the `02-client` submodule to call into a light client module. A light client module developer is only required to implement a set interfaces as defined in the `modules/core/exported` package of ibc-go.

A light client module developer should be concerned with three main interfaces:

- [`ClientState`](#clientstate) encapsulates the light client implementation and its semantics.
- [`ConsensusState`](#consensusstate) tracks consensus data used for verification of client updates, misbehaviour detection and proof verification of counterparty state.
- [`ClientMessage`](#clientmessage) used for submitting block headers for client updates and submission of misbehaviour evidence using conflicting headers.

Throughout this guide the `07-tendermint` light client module may be referred to as a reference example.

## Concepts and vocabulary

### `ClientState`

`ClientState` is a term used to define the data structure which encapsulates opaque light client state. The `ClientState` contains all the information needed to verify a `ClientMessage` and perform membership and non-membership proof verification of counterparty state. This includes properties that refer to the remote state machine, the light client type and the specific light client instance.

For example:

- Constraints used for client updates.
- Constraints used for misbehaviour detection.
- Constraints used for state verification.
- Constraints used for client upgrades.

The `ClientState` type maintained within the light client module *must* implement the [`ClientState`](https://github.com/cosmos/ibc-go/tree/02-client-refactor-beta1/modules/core/exported/client.go#L36) interface defined in `core/modules/exported/client.go`.
The methods which make up this interface are detailed at a more granular level in the [ClientState section of this guide](02-client-state.md).

Please refer to the `07-tendermint` light client module's [`ClientState` definition](https://github.com/cosmos/ibc-go/tree/02-client-refactor-beta1/proto/ibc/lightclients/tendermint/v1/tendermint.proto#L18) containing information such as chain ID, status, latest height, unbonding period and proof specifications.

### `ConsensusState`

`ConsensusState` is a term used to define the data structure which encapsulates consensus data at a particular point in time, i.e. a unique height or sequence number of a state machine. There must exist a single trusted `ConsensusState` for each height. `ConsensusState` generally contains a trusted root, validator set information and timestamp.

For example, the `ConsensusState` of the `07-tendermint` light client module defines a trusted root which is used by the `ClientState` to perform verification of membership and non-membership commitment proofs, as well as the next validator set hash used for verifying headers can be trusted in client updates.

The `ConsensusState` type maintained within the light client module *must* implement the [`ConsensusState`](https://github.com/cosmos/ibc-go/tree/02-client-refactor-beta1/modules/core/exported/client.go#L134) interface defined in `modules/core/exported/client.go`.
The methods which make up this interface are detailed at a more granular level in the [`ConsensusState` section of this guide](03-consensus-state.md).

### `Height`

`Height` defines a monotonically increasing sequence number which provides ordering of consensus state data persisted through client updates.
IBC light client module developers are expected to use the [concrete type](https://github.com/cosmos/ibc-go/tree/02-client-refactor-beta1/proto/ibc/core/client/v1/client.proto#L89) provided by the `02-client` submodule. This implements the expectations required by the [`Height`](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/core/exported/client.go#L156) interface defined in `modules/core/exported/client.go`.

### `ClientMessage`

`ClientMessage` refers to the interface type [`ClientMessage`](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/core/exported/client.go#L147) used for performing updates to a `ClientState` stored on chain.
This may be any concrete type which produces a change in state to the IBC client when verified.

The following are considered as valid update scenarios:

- A block header which when verified inserts a new `ConsensusState` at a unique height.
- A batch of block headers which when verified inserts `N` `ConsensusState` instances for `N` unique heights.
- Evidence of misbehaviour provided by two conflicting block headers.

Learn more in the [Handling update and misbehaviour](04-updates-and-misbehaviour.md) section.
