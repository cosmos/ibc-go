# Migrating from ibc-go v3 to v4

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

- No relevant changes were made in this release.

## IBC Apps

### ICS03 - Connection

Crossing hellos have been removed from 03-connection handshake negotiation. 
`PreviousConnectionId` in `MsgConnectionOpenTry` has been deprecated and is no longer used by core IBC.

### ICS04 - Channel 

The `WriteAcknowledgement` API now takes the `exported.Acknowledgement` type instead of passing in the acknowledgement byte array directly. 
This is an API breaking change and as such IBC application developers will have to update any calls to `WriteAcknowledgement`. 

The `OnChanOpenInit` application callback has been modified.
The return signature now includes the application version as detailed in the latest IBC [spec changes](https://github.com/cosmos/ibc/pull/629).

The `NewErrorAcknowledgement` method signature has changed.
It now accepts an `error` rather than a `string`. This was done in order to prevent accidental state changes.
All error acknowledgements now contain a deterministic ABCI code and error message. It is the responsibility of the application developer to emit error details in events.

Crossing hellos have been removed from 04-channel handshake negotiation. 
IBC Applications no longer need to account from already claimed capabilities in the `OnChanOpenTry` callback. The capability provided by core IBC must be able to be claimed with error. 
`PreviousChannelId` in `MsgChannelOpenTry` has been deprecated and is no longer used by core IBC.

### ICS27 - Interchain Accounts

The `RegisterInterchainAccount` API has been modified to include an additional `version` argument. This change has been made in order to support ICS29 fee middleware, for relayer incentivization of ICS27 packets.
Consumers of the `RegisterInterchainAccount` are now expected to build the appropriate JSON encoded version string themselves and pass it accordingly. 
This should be constructed within the interchain accounts authentication module which leverages the APIs exposed via the interchain accounts `controllerKeeper`. If an empty string is passed in the `version` argument, then the version will be initialized to a default value in the `OnChanOpenInit` callback of the controller's handler, so that channel handshake can proceed.

The following code snippet illustrates how to construct an appropriate interchain accounts `Metadata` and encode it as a JSON bytestring:

```go
icaMetadata := icatypes.Metadata{
    Version:                icatypes.Version,
    ControllerConnectionId: controllerConnectionID,
    HostConnectionId:       hostConnectionID,
    Encoding:               icatypes.EncodingProtobuf,
    TxType:                 icatypes.TxTypeSDKMultiMsg,
}

appVersion, err := icatypes.ModuleCdc.MarshalJSON(&icaMetadata)
if err != nil {
    return err
}

if err := k.icaControllerKeeper.RegisterInterchainAccount(ctx, msg.ConnectionId, msg.Owner, string(appVersion)); err != nil {
    return err
}
```

Similarly, if the application stack is configured to route through ICS29 fee middleware and a fee enabled channel is desired, construct the appropriate ICS29 `Metadata` type:

```go
icaMetadata := icatypes.Metadata{
    Version:                icatypes.Version,
    ControllerConnectionId: controllerConnectionID,
    HostConnectionId:       hostConnectionID,
    Encoding:               icatypes.EncodingProtobuf,
    TxType:                 icatypes.TxTypeSDKMultiMsg,
}

appVersion, err := icatypes.ModuleCdc.MarshalJSON(&icaMetadata)
if err != nil {
    return err
}

feeMetadata := feetypes.Metadata{
    AppVersion: string(appVersion),
    FeeVersion: feetypes.Version,
}

feeEnabledVersion, err := feetypes.ModuleCdc.MarshalJSON(&feeMetadata)
if err != nil {
    return err
}

if err := k.icaControllerKeeper.RegisterInterchainAccount(ctx, msg.ConnectionId, msg.Owner, string(feeEnabledVersion)); err != nil {
    return err
}
```

## Relayers

When using the `DenomTrace` gRPC, the full IBC denomination with the `ibc/` prefix may now be passed in.

Crossing hellos are no longer supported by core IBC for 03-connection and 04-channel. The handshake should be completed in the logical 4 step process (INIT, TRY, ACK, CONFIRM).
