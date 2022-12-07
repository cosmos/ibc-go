<!--
order: 1
-->

# Overview 

Learn how to build IBC lightclient modules and fulfill the interfaces required to integrate with core IBC. {synopsis}

The following aims to provide a high level IBC lightclient module developer guide. The lifecycle of IBC lightclients are created and maintained by the core IBC `MsgServer` and the abstractions provided by the `02-client` submodule. A lightclient module developer is only required to implement the methods and fulfil the expectations of a defined set of interfaces. In ibc-go these interfaces reside within the `core/modules/exported` package. 

Throughout this guide the `07-tendermint` lightclient module may be referred to as a reference example.

## Concepts and vocabulary

This guide serves to be implementation specific with respect to ibc-go. Readers are expected to familiarize themselves with the IBC protocol specifications.
Please refer to the defintions outlined in [ICS-002 Client Semantics](https://github.com/cosmos/ibc/tree/main/spec/core/ics-002-client-semantics#Definitions).

### ClientState 

ClientState is a term used to define the data structure which encapsulates opaque lightclient state. This refers to internal data that accommodate the semantics of a lightclient. This may be any arbitrary data such as:

- Constraints used for client updates.
- Constraints used for misbehaviour detection.
- Constraints used for state verification.
- Constraints used for client upgrades.

The `ClientState` type maintained within the lightclient module *must* implement the `ClientState` interface defined in `core/modules/exported/client.go`.
The methods which make up this interface are detailed at a more granular level in the [ClientState section of this guide](./client-state.md).

For reference, see the `07-tendermint` lightclient module's [`ClientState` defintion](https://github.com/cosmos/ibc-go/blob/v6.0.0-rc1/proto/ibc/lightclients/tendermint/v1/tendermint.proto#L18). 

### ConsensusState

ConsensusState is a term used to define the data structure which encapsulates consensus data at a particular point in time, i.e. a unique height or sequence number of a state machine. There must exist a single trusted `ConsensusState` for each height. `ConsensusState` generally contains a trusted root, validator set information and timestamp. 

For example, the `ConsensusState` of the `07-tendermint` lightclient module defines a trusted root is used by the `ClientState` to perform verification of membership and non-membership commitment proofs, as well as the next validator set hash used for verifying headers can be trusted in client updates. 

The `ConsensusState` type maintained within the lightclient module *must* implement the `ConsensusState` interface defined in `core/modules/exported/client.go`.
The methods which make up this interface are detailed at a more granular level in the [ConsensusState section of this guide](./consensus-state.md).

### Height

Height defines a monotonically increasing sequence number which provides ordering of consensus state data persisted through updates. 
IBC lightclient module developers are expected to use the concrete type provided by the `02-client` submodule. This implements the expectations required by the `Height` interface in `core/modules/exported/client.go`.

