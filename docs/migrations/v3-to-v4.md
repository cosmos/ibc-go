# Migrating from ibc-go v2 to v3

This document is intended to highlight significant changes which may require more information than presented in the CHANGELOG.
Any changes that must be done by a user of ibc-go should be documented here.

There are four sections based on the four potential user groups of this document:
- Chains
- IBC Apps
- Relayers
- IBC Light Clients

**Note:** ibc-go supports golang semantic versioning and therefore all imports must be updated to bump the version number on major releases.
```go
github.com/cosmos/ibc-go/v3 -> github.com/cosmos/ibc-go/v4
```

No genesis or in-place migrations required when upgrading from v1 or v2 of ibc-go.

## Chains

### IS04 - Channel

The `WriteAcknowledgement` API now takes the `exported.Acknowledgement` type instead of passing in the acknowledgement byte array directly.
This is an API breaking change and as such IBC application developers will have to update any calls to `WriteAcknowledgement`.

## IBC Light Clients

### `ClientState` interface changes

The `VerifyUpgradeAndUpdateState` function has been modified. The client state and consensus state return values have been removed.

Light clients **must** handle all management of client and consensus states including the setting of updated client state and consensus state in the client store.

The `CheckHeaderAndUpdateState` function has been split into 4 new functions:

- `VerifyClientMessage` verifies a `ClientMessage`. A `ClientMessage` could be a `Header`, `Misbehaviour`, or batch update. Calls to `CheckForMisbehaviour`, `UpdateState`, and `UpdateStateOnMisbehaviour` will assume that the content of the `ClientMessage` has been verified and can be trusted. An error should be returned if the `ClientMessage` fails to verify.

- `CheckForMisbehaviour` checks for evidence of a misbehaviour in `Header` or `Misbehaviour` types.

- `UpdateStateOnMisbehaviour` performs appropriate state changes on a `ClientState` given that misbehaviour has been detected and verified.

- `UpdateState` updates and stores as necessary any associated information for an IBC client, such as the `ClientState` and corresponding `ConsensusState`. An error is returned if `ClientMessage` is of type `Misbehaviour`. Upon successful update, a list containing the updated consensus state height is returned.

The `CheckMisbehaviourAndUpdateState` function has been removed from `ClientState` interface. This functionality is now encapsulated by the usage of `VerifyClientMessage`, `CheckForMisbehaviour`, `UpdateStateOnMisbehaviour`, `UpdateState`.

The function `GetTimestampAtHeight` has been added to the `ClientState` interface. It should return the timestamp for a consensus state associated with the provided height.

### `Header` and `Misbehaviour`

`exported.Header` and `exported.Misbehaviour` interface types have been merged and renamed to `ClientMessage` interface.

`GetHeight` function has been removed from `exported.Header` and thus is not included in the `ClientMessage` interface

### `ConsensusState`

The `GetRoot` function has been removed from consensus state interface since it was not used by core IBC.

### Light client implementations

09-localhost light client implementation has been removed because it is currently non-functional.

### Client Keeper

Keeper function `CheckMisbehaviourAndUpdateState` has been removed since function `UpdateClient` can now handle updating `ClientState` on `ClientMessage` type which can be any `Misbehaviour` implementations.  

### SDK Message

`MsgSubmitMisbehaviour` is deprecated since `MsgUpdateClient` can now submit a `ClientMessage` type which can be any `Misbehaviour` implementations.

The field `header` in `MsgUpdateClient` has been renamed to `client message`.
