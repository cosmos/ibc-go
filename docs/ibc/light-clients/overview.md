<!--
order: 1
-->

# Overview 

Learn how to build IBC light client modules and fulfill the interfaces required to integrate with core IBC. {synopsis}

<!-- Add prerequisite readingn section? -->
## Pre-requisites Readings

- [IBC Overview](../overview.md)) {prereq}
- [IBC Transport, Authentication, and Ordering Layer - Clients](https://tutorials.cosmos.network/academy/3-ibc/4-clients.html) {prereq}
- [ICS-002 Client Semantics](https://github.com/cosmos/ibc/tree/main/spec/core/ics-002-client-semantics) {prereq}

The following aims to provide a high level IBC light client module developer guide. Access to IBC light clients are gated by the core IBC `MsgServer` which utilizes the abstractions set by the `02-client` submodule to call into a light client module. A light client module developer is only required to implement a set interfaces as defined in the `core/modules/exported` package of ibc-go. 

A light client module developer should be concerned with three main interfaces:

- [`ClientState`](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L36) encapsulates the light client implementation and its semantics.
- [`ConsensusState`](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L134) tracks critical consensus data used for verification of client updates and proof verification of counterparty state.
- [`ClientMessage`](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L148) used for submitting block headers for client updates and submission of misbehaviour evidence using conflicting headers. 

Throughout this guide the `07-tendermint` light client module may be referred to as a reference example.

## Concepts and vocabulary

### ClientState 

ClientState is a term used to define the data structure which encapsulates opaque light client state. This refers to internal data that form the rules concerning trust gaurantees and proof verification. This may be any arbitrary data such as:

- Constraints used for client updates.
- Constraints used for misbehaviour detection.
- Constraints used for state verification.
- Constraints used for client upgrades.

The `ClientState` type maintained within the light client module *must* implement the [`ClientState`]((https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L36)) interface defined in `core/modules/exported/client.go`.
The methods which make up this interface are detailed at a more granular level in the [ClientState section of this guide](./client-state.md).

For reference, see the `07-tendermint` light client module's [`ClientState` defintion](https://github.com/cosmos/ibc-go/blob/v6.0.0-rc1/proto/ibc/lightclients/tendermint/v1/tendermint.proto#L18). 

### ConsensusState

ConsensusState is a term used to define the data structure which encapsulates consensus data at a particular point in time, i.e. a unique height or sequence number of a state machine. There must exist a single trusted `ConsensusState` for each height. `ConsensusState` generally contains a trusted root, validator set information and timestamp. 

For example, the `ConsensusState` of the `07-tendermint` light client module defines a trusted root is used by the `ClientState` to perform verification of membership and non-membership commitment proofs, as well as the next validator set hash used for verifying headers can be trusted in client updates. 

The `ConsensusState` type maintained within the light client module *must* implement the [`ConsensusState`](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L134) interface defined in `core/modules/exported/client.go`.
The methods which make up this interface are detailed at a more granular level in the [ConsensusState section of this guide](./consensus-state.md).

### Height

Height defines a monotonically increasing sequence number which provides ordering of consensus state data persisted through client updates. 
IBC light client module developers are expected to use the concrete type provided by the `02-client` submodule. This implements the expectations required by the `Height` interface defined in `core/modules/exported/client.go`.

### ClientMessage

ClientMessage refers to the interface type [`ClientMessage`](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L148) used for performing updates to a `ClientState` stored on chain. 
This may be any concrete type which produces a change in state to the IBC client when verified.

The following are considered as valid update scenarios:

- A block header which when verified inserts a new `ConsensusState` at a unique height. 
- A batch of block headers which when verified inserts `N` `ConsensusState` instances for `N` unique heights.
- Evidence of misbehaviour provided by two conflicting block headers.

Learn more in the [handling client updates](./update.md) and [handling misbehaviour](./misbehaviour.md) sections. 
