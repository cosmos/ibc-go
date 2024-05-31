---
title: Keeper API
sidebar_label: Keeper API
sidebar_position: 3
slug: /apps/interchain-accounts/legacy/keeper-api
---


# Keeper API

## Deprecation Notice

**This document is deprecated and will be removed in future releases**.

The controller submodule keeper exposes two legacy functions that allow respectively for custom authentication modules to register interchain accounts and send packets to the interchain account.

## `RegisterInterchainAccount`

The authentication module can begin registering interchain accounts by calling `RegisterInterchainAccount`:

```go
if err := keeper.icaControllerKeeper.RegisterInterchainAccount(ctx, connectionID, owner.String(), version); err != nil {
    return err
}

return nil
```

The `version` argument is used to support ICS-29 fee middleware for relayer incentivization of ICS-27 packets. Consumers of the `RegisterInterchainAccount` are expected to build the appropriate JSON encoded version string themselves and pass it accordingly. If an empty string is passed in the `version` argument, then the version will be initialized to a default value in the `OnChanOpenInit` callback of the controller's handler, so that channel handshake can proceed.

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

if err := keeper.icaControllerKeeper.RegisterInterchainAccount(ctx, controllerConnectionID, owner.String(), string(appVersion)); err != nil {
    return err
}
```

Similarly, if the application stack is configured to route through ICS-29 fee middleware and a fee enabled channel is desired, construct the appropriate ICS-29 `Metadata` type:

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

if err := keeper.icaControllerKeeper.RegisterInterchainAccount(ctx, controllerConnectionID, owner.String(), string(feeEnabledVersion)); err != nil {
    return err
}
```

## `SendTx`

The authentication module can attempt to send a packet by calling `SendTx`:

```go
// Authenticate owner
// perform custom logic
    
// Construct controller portID based on interchain account owner address
portID, err := icatypes.NewControllerPortID(owner.String())
if err != nil {
    return err
}
    
// Obtain data to be sent to the host chain. 
// In this example, the owner of the interchain account would like to send a bank MsgSend to the host chain. 
// The appropriate serialization function should be called. The host chain must be able to deserialize the transaction. 
// If the host chain is using the ibc-go host module, `SerializeCosmosTx` should be used. 
msg := &banktypes.MsgSend{FromAddress: fromAddr, ToAddress: toAddr, Amount: amt}
data, err := icatypes.SerializeCosmosTx(keeper.cdc, []proto.Message{msg})
if err != nil {
    return err
}

// Construct packet data
packetData := icatypes.InterchainAccountPacketData{
    Type: icatypes.EXECUTE_TX,
    Data: data,
}

// Obtain timeout timestamp
// An appropriate timeout timestamp must be determined based on the usage of the interchain account.
// If the packet times out, the channel will be closed requiring a new channel to be created.
timeoutTimestamp := obtainTimeoutTimestamp()

// Send the interchain accounts packet, returning the packet sequence
// A nil channel capability can be passed, since the controller submodule (and not the authentication module) 
// claims the channel capability since ibc-go v6.
seq, err = keeper.icaControllerKeeper.SendTx(ctx, nil, portID, packetData, timeoutTimestamp)
```

The data within an `InterchainAccountPacketData` must be serialized using a format supported by the host chain. 
If the host chain is using the ibc-go host chain submodule, `SerializeCosmosTx` should be used. If the `InterchainAccountPacketData.Data` is serialized using a format not supported by the host chain, the packet will not be successfully received.  
