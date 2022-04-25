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

### ClientState interface changes

The `VerifyUpgradeAndUpdateState` function has been modified. The client state and consensus state return value has been removed. 

Light clients **must** handle all management of client and consensus states including the setting of updated client state and consensus state in the client store.

The `CheckHeaderAndUpdateState` function has been split into 4 new functions: `VerifyClientMessage`, `CheckForMisbehaviour`, `UpdateState`, 
`UpdateStateOnMisbehaviour`

Light client implementations now need to manage setting of client and consensus states for these interface functions `UpdateState`, `UpdateStateOnMisbehaviour`, `VerifyUpgradeAndUpdateState`, `CheckSubstituteAndUpdateState`

The `CheckMisbehaviourAndUpdateState` function has been removed from `ClientState` interface

The `GetTimestampAtHeight` has been added to the `ClientState` interface

### Header and Misbehaviour

`exported.Header` and `exported.Misbehaviour` interface types have been merged and renamed to `ClientMessage` interface

### ConsensusState

The `GetRoot` function has been removed from consensus state interface.