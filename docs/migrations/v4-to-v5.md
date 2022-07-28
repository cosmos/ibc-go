# Migrating from v4 to v5

This document is intended to highlight significant changes which may require more information than presented in the CHANGELOG.
Any changes that must be done by a user of ibc-go should be documented here.

There are four sections based on the four potential user groups of this document:
- Chains
- IBC Apps
- Relayers
- IBC Light Clients

**Note:** ibc-go supports golang semantic versioning and therefore all imports must be updated to bump the version number on major releases.
```go
github.com/cosmos/ibc-go/v4 -> github.com/cosmos/ibc-go/v5
```

No genesis or in-place migrations required when upgrading from v1 or v2 of ibc-go.

## Chains

### Ante decorator

The `AnteDecorator` type in `core/ante` has been renamed to `RedundantRelayDecorator` (and the corresponding constructor function to `NewRedundantRelayDecorator`).

## IBC Apps

### Core

The `key` parameter of the `NewKeeper` functions in `modules/core/keeper` is now of type `storetypes.StoreKey` (`storetypes "github.com/cosmos/cosmos-sdk/store/types"`).

The `RegisterRESTRoutes` function in `modules/core` has been removed.

### ICS03 - Connection

The `key` parameter of the `NewKeeper` functions in `modules/core/03-connection/keeper` is now of type `storetypes.StoreKey` (`storetypes "github.com/cosmos/cosmos-sdk/store/types"`).

### ICS04 - Channel 

The function `NewPacketId` in `modules/core/04-channel/types` has been renamed to `NewPacketID`.

The `key` parameter of the `NewKeeper` functions in `modules/core/04-channel/keeper` is now of type `storetypes.StoreKey` (`storetypes "github.com/cosmos/cosmos-sdk/store/types"`).

### ICS20 - Transfer

The `key` parameter of the `NewKeeper` function in `modules/apps/transfer/keeper` is now of type `storetypes.StoreKey` (`storetypes "github.com/cosmos/cosmos-sdk/store/types"`).

The `amount` parameter of function `GetTransferCoin` in `modules/apps/transfer/types` is now of type `math.Int` (`"cosmossdk.io/math"`).

The `RegisterRESTRoutes` function in `modules/apps/transfer` has been removed.

### ICS27 - Interchain Accounts

The `key` parameter of the `NewKeeper` functions in 

- `modules/apps/27-interchain-accounts/controller/keeper` 
- and `modules/apps/27-interchain-accounts/host/keeper` 

The `RegisterRESTRoutes` function in `modules/apps/27-interchain-accounts` has been removed.

The response of a message execution on the host chain is constructed now like this:

```
&codectypes.Any{
  TypeUrl: sdk.MsgTypeURL(msg),
  Value:   msgResponse,
}
```

See [ADR-03](../architecture/adr-003-ics27-acknowledgement.md/#next-major-version-format) for more information.

### ICS29 - Fee Middleware

The `key` parameter of the `NewKeeper` function in `modules/apps/29-fee` is now of type `storetypes.StoreKey` (`storetypes "github.com/cosmos/cosmos-sdk/store/types"`).

The `RegisterRESTRoutes` function in `modules/apps/29-fee` has been removed.

### IBC testing package

The `MockIBCApp` type has been renamed to `IBCApp` (and the corresponding constructor function to `NewIBCApp`). This has resulted therefore in:
- The `IBCApp` field of the `*IBCModule` in `testing/mock` to change its type as well to `*IBCApp`.
- The `app` parameter to `*NewIBCModule` in `testing/mock` to change its type as well to `*IBCApp`.

The `MockEmptyAcknowledgement` field has been renamed to `EmptyAcknowledgement` (and the corresponding constructor function to `NewEmptyAcknowledgement`).

The return type of the function `LastCommitID` of the `TestingApp` interface in `testing` has changed to `storetypes.CommitID` (`storetypes "github.com/cosmos/cosmos-sdk/store/types"`).

The `powerReduction` parameter of the function `SetupWithGenesisValSet` in `testing` is now of type `math.Int` (`"cosmossdk.io/math"`).

The `accAmt` parameter of the functions

- `AddTestAddrsFromPubKeys` ,
- `AddTestAddrs`
- and `AddTestAddrsIncremental`

in `testing/simapp` are now of type `math.Int` (`"cosmossdk.io/math"`).

The `RegisterRESTRoutes` function in `testing/mock` has been removed.

## Relayers

- No relevant changes were made in this release.

## IBC Light Clients

### ICS02 - Client

The `key` parameter of the `NewKeeper` function in `modules/core/02-client/keeper` is now of type `storetypes.StoreKey` (`storetypes "github.com/cosmos/cosmos-sdk/store/types"`).
