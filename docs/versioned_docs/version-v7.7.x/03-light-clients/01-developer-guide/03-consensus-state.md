---
title: Consensus State interface
sidebar_label: Consensus State interface
sidebar_position: 3
slug: /ibc/light-clients/consensus-state
---


# Implementing the `ConsensusState` interface

A `ConsensusState` is the snapshot of the counterparty chain, that an IBC client uses to verify proofs (e.g. a block). 

The further development of multiple types of IBC light clients and the difficulties presented by this generalization problem (see [ADR-006](https://github.com/cosmos/ibc-go/blob/main/docs/architecture/adr-006-02-client-refactor.md) for more information about this historical context) led to the design decision of each client keeping track of and set its own `ClientState` and `ConsensusState`, as well as the simplification of client `ConsensusState` updates through the generalized `ClientMessage` interface.

The below [`ConsensusState`](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L134) interface is a generalized interface for the types of information a `ConsensusState` could contain. For a reference `ConsensusState` implementation, please see the [Tendermint light client `ConsensusState`](https://github.com/cosmos/ibc-go/blob/main/modules/light-clients/07-tendermint/consensus_state.go).

## `ClientType` method

This is the type of client consensus. It should be the same as the `ClientType` return value for the [corresponding `ClientState` implementation](02-client-state.md).

## `GetTimestamp` method

`GetTimestamp` should return the timestamp (in nanoseconds) of the consensus state snapshot.

## `ValidateBasic` method

`ValidateBasic` should validate every consensus state field and should return an error if any value is invalid. The light client implementer is in charge of determining which checks are required.
