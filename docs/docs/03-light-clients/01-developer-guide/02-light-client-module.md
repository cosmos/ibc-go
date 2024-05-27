---
title: Light Client Module interface
sidebar_label: Light Client Module interface
sidebar_position: 2
slug: /ibc/light-clients/light-client-module
---


# Implementing the `LightClientModule` interface

## `RegisterStoreProvider` method

`RegisterStoreProvider` is called by core IBC when a `LightClientModule` is added to the router. It allows the `LightClientModule` to set a `ClientStoreProvider` ([interface defined in `modules/core/exported`](https://github.com/cosmos/ibc-go/blob/06fd8eb5ee1697e3b43be7528a6e42f5e4a4613c/modules/core/exported/client.go#L49-L52)) which supplies isolated prefix client stores to IBC light client instances.

## `Status` method

`Status` must return the status of the client.

- An `Active` status indicates that clients are allowed to process packets.
- A `Frozen` status indicates that misbehaviour was detected in the counterparty chain and the client is not allowed to be used.
- An `Expired` status indicates that a client is not allowed to be used because it was not updated for longer than the trusting period.
- An `Unknown` status indicates that there was an error in determining the status of a client.

All possible `Status` types can be found [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/core/exported/client.go#L22-L32).

This field is returned in the response of the gRPC [`ibc.core.client.v1.Query/ClientStatus`](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/core/02-client/types/query.pb.go#L665) endpoint.

## `TimestampAtHeight` method

`TimestampAtHeight` must return the timestamp for the consensus state associated with the provided height.
This value is used to facilitate timeouts by checking the packet timeout timestamp against the returned value.

## `LatestHeight` method

`LatestHeight` should return the latest block height that the client state represents.

## `Initialize` method

Clients must validate the initial consensus state, and set the initial client state and consensus state in the provided client store.
Clients may also store any necessary client-specific metadata.

`Initialize` is called when a [client is created](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/core/02-client/keeper/client.go#L30).

## `UpdateState` method

`UpdateState` updates and stores as necessary any associated information for an IBC client, such as the `ClientState` and corresponding `ConsensusState`. See section [`UpdateState`](05-updates-and-misbehaviour.md#updatestate) for more information.

## `UpdateStateOnMisbehaviour` method

`UpdateStateOnMisbehaviour` should perform appropriate state changes on a client state given that misbehaviour has been detected and verified. See section [`UpdateStateOnMisbehaviour`](05-updates-and-misbehaviour.md#updatestateonmisbehaviour) for more information.

## `VerifyMembership` method

`VerifyMembership` must verify the existence of a value at a given commitment path at the specified height. For more information about membership proofs
see the [Existence and non-existence proofs section](07-proofs.md).

## `VerifyNonMembership` method

`VerifyNonMembership` must verify the absence of a value at a given commitment path at a specified height. For more information about non-membership proofs
see the [Existence and non-existence proofs section](07-proofs.md).

## `VerifyClientMessage` method

`VerifyClientMessage` must verify a `ClientMessage`. A `ClientMessage` could be a `Header`, `Misbehaviour`, or batch update.
It must handle each type of `ClientMessage` appropriately. Calls to `CheckForMisbehaviour`, `UpdateState`, and `UpdateStateOnMisbehaviour`
will assume that the content of the `ClientMessage` has been verified and can be trusted. An error should be returned
if the ClientMessage fails to verify. See section [`VerifyClientMessage`](05-updates-and-misbehaviour.md#verifyclientmessage) for more information.

## `CheckForMisbehaviour` method

Checks for evidence of a misbehaviour in `Header` or `Misbehaviour` type. It assumes the `ClientMessage`
has already been verified. See section [`CheckForMisbehaviour`](05-updates-and-misbehaviour.md#checkformisbehaviour) for more information.

## `RecoverClient` method

`RecoverClient` is used to recover an expired or frozen client by updating the client with the state of a substitute client. The method must verify that the provided substitute may be used to update the subject client. See section [Implementing `RecoverClient`](./08-proposals.md#implementing-recoverclient) for more information.

## `VerifyUpgradeAndUpdateState` method

`VerifyUpgradeAndUpdateState` provides a path to upgrading clients given an upgraded `ClientState`, upgraded `ConsensusState` and proofs for each. See section [Implementing `VerifyUpgradeAndUpdateState`](./06-upgrades.md#implementing-verifyupgradeandupdatestate) for more information.
