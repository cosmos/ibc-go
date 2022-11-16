# Migrating from ibc-go v5 to v6

This document is intended to highlight significant changes which may require more information than presented in the CHANGELOG.
Any changes that must be done by a user of ibc-go should be documented here.

There are four sections based on the four potential user groups of this document:
- Chains
- IBC Apps
- Relayers
- IBC Light Clients

**Note:** ibc-go supports golang semantic versioning and therefore all imports must be updated to bump the version number on major releases.

## Chains

- No relevant changes were made in this release.

## IBC Apps

- No relevant changes were made in this release.

## Relayers

- No relevant changes were made in this release.

## IBC Light Clients

### `ClientState` interface changes

The `VerifyUpgradeAndUpdateState` function has been modified. The client state and consensus state return values have been removed.

Light clients **must** handle all management of client and consensus states including the setting of updated client state and consensus state in the client store.

The `CheckHeaderAndUpdateState` function has been split into 4 new functions:

- `VerifyClientMessage` verifies a `ClientMessage`. A `ClientMessage` could be a `Header`, `Misbehaviour`, or batch update. Calls to `CheckForMisbehaviour`, `UpdateState`, and `UpdateStateOnMisbehaviour` will assume that the content of the `ClientMessage` has been verified and can be trusted. An error should be returned if the `ClientMessage` fails to verify.

- `CheckForMisbehaviour` checks for evidence of a misbehaviour in `Header` or `Misbehaviour` types.

- `UpdateStateOnMisbehaviour` performs appropriate state changes on a `ClientState` given that misbehaviour has been detected and verified.

- `UpdateState` updates and stores as necessary any associated information for an IBC client, such as the `ClientState` and corresponding `ConsensusState`. An error is returned if `ClientMessage` is of type `Misbehaviour`. Upon successful update, a list containing the updated consensus state height is returned.

The `CheckMisbehaviourAndUpdateState` function has been removed from `ClientState` interface. This functionality is now encapsulated by the usage of `VerifyClientMessage`, `CheckForMisbehaviour`, `UpdateStateOnMisbehaviour`.

The function `GetTimestampAtHeight` has been added to the `ClientState` interface. It should return the timestamp for a consensus state associated with the provided height.

A zero proof height is now allowed by core IBC and may be passed into `VerifyMembership` and `VerifyNonMembership`. Light clients are responsible for returning an error if a zero proof height is invalid behaviour. 

### `Header` and `Misbehaviour`

`exported.Header` and `exported.Misbehaviour` interface types have been merged and renamed to `ClientMessage` interface.

`GetHeight` function has been removed from `exported.Header` and thus is not included in the `ClientMessage` interface

### `ConsensusState`

The `GetRoot` function has been removed from consensus state interface since it was not used by core IBC.

### Light client implementations

The `09-localhost` light client implementation has been removed because it is currently non-functional.

An upgrade handler has been added to supply chain developers with the logic needed to prune the ibc client store and successfully complete the removal of `09-localhost`.
Add the following to the application upgrade handler in `app/app.go`, calling `MigrateToV6` to perform store migration logic.

```go
import (
    // ...
    ibcv6 "github.com/cosmos/ibc-go/v6/modules/core/migrations/v6"
)

// ...

app.UpgradeKeeper.SetUpgradeHandler(
    upgradeName,
    func(ctx sdk.Context, _ upgradetypes.Plan, _ module.VersionMap) (module.VersionMap, error) {
        // prune the 09-localhost client from the ibc client store
        ibcv6.MigrateToV6(ctx, app.IBCKeeper.ClientKeeper)

        return app.mm.RunMigrations(ctx, app.configurator, fromVM)
    },
)
```

Please note the above upgrade handler is optional and should only be run if chains have an existing `09-localhost` client stored in state.
A simple query can be performed to check for a `09-localhost` client on chain.

For example:

```
simd query ibc client states | grep 09-localhost
```

### Client Keeper

Keeper function `CheckMisbehaviourAndUpdateState` has been removed since function `UpdateClient` can now handle updating `ClientState` on `ClientMessage` type which can be any `Misbehaviour` implementations.  

### SDK Message

`MsgSubmitMisbehaviour` is deprecated since `MsgUpdateClient` can now submit a `ClientMessage` type which can be any `Misbehaviour` implementations.

The field `header` in `MsgUpdateClient` has been renamed to `client_message`.
