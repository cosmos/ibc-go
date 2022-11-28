<!--
order: 3
-->

# Keeper API

## Deprecation Notice

**This document is deprecated and will be removed in future releases**.

The controller submodule keeper exposes legacy two functions that allow custom authentication modules to register interchain accounts and send packets to the interchain account.

## `RegisterInterchainAccount`

The authentication module can begin registering interchain accounts by calling `RegisterInterchainAccount`:

```go
if err := keeper.icaControllerKeeper.RegisterInterchainAccount(ctx, connectionID, owner.String(), version); err != nil {
    return err
}

return nil
```

The `version` argument is used to support ICS29 fee middleware for relayer incentivization of ICS-27 packets. Consumers of the `RegisterInterchainAccount` are expected to build the appropriate JSON encoded version string themselves and pass it accordingly. If an empty string is passed in the `version` argument, then the version will be initialized to a default value in the `OnChanOpenInit` callback of the controller's handler, so that channel handshake can proceed.

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

if err := keeper.icaControllerKeeper.RegisterInterchainAccount(ctx, controllerConnectionID, owner.String(), string(feeEnabledVersion)); err != nil {
    return err
}
```

### `SendTx`

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
If the host chain is using the ibc-go host chain submodule, `SerializeCosmosTx` should be used. If the `InterchainAccountPacketData.Data` is serialized using a format not support by the host chain, the packet will not be successfully received.  

### `OnAcknowledgementPacket`

Controller chains will be able to access the acknowledgement written into the host chain state once a relayer relays the acknowledgement. 
The acknowledgement bytes will be passed to the auth module via the `OnAcknowledgementPacket` callback. 
Auth modules are expected to know how to decode the acknowledgement. 

If the controller chain is connected to a host chain using the host module on ibc-go, it may interpret the acknowledgement bytes as follows:

Begin by unmarshaling the acknowledgement into `sdk.TxMsgData`:

```go
var ack channeltypes.Acknowledgement
if err := channeltypes.SubModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
    return err
}

txMsgData := &sdk.TxMsgData{}
if err := proto.Unmarshal(ack.GetResult(), txMsgData); err != nil {
    return err
}
```

If the `txMsgData.Data` field is non nil, the host chain is using SDK version <= v0.45. 
The auth module should interpret the `txMsgData.Data` as follows:

```go
switch len(txMsgData.Data) {
case 0:
    // see documentation below for SDK 0.46.x or greater
default:
    for _, msgData := range txMsgData.Data {
        if err := handler(msgData); err != nil {
            return err
        }
    }
...
}            
```

A handler will be needed to interpret what actions to perform based on the message type sent.
A router could be used, or more simply a switch statement.

```go
func handler(msgData sdk.MsgData) error {
switch msgData.MsgType {
case sdk.MsgTypeURL(&banktypes.MsgSend{}):
    msgResponse := &banktypes.MsgSendResponse{}
    if err := proto.Unmarshal(msgData.Data, msgResponse}; err != nil {
        return err
    }

    handleBankSendMsg(msgResponse)

case sdk.MsgTypeURL(&stakingtypes.MsgDelegate{}):
    msgResponse := &stakingtypes.MsgDelegateResponse{}
    if err := proto.Unmarshal(msgData.Data, msgResponse}; err != nil {
        return err
    }

    handleStakingDelegateMsg(msgResponse)

case sdk.MsgTypeURL(&transfertypes.MsgTransfer{}):
    msgResponse := &transfertypes.MsgTransferResponse{}
    if err := proto.Unmarshal(msgData.Data, msgResponse}; err != nil {
        return err
    }

    handleIBCTransferMsg(msgResponse)
 
default:
    return
}
```

If the `txMsgData.Data` is empty, the host chain is using SDK version > v0.45.
The auth module should interpret the `txMsgData.Responses` as follows:

```go
...
// switch statement from above
case 0:
    for _, any := range txMsgData.MsgResponses {
        if err := handleAny(any); err != nil {
            return err
        }
    }
}
``` 

A handler will be needed to interpret what actions to perform based on the type URL of the Any. 
A router could be used, or more simply a switch statement. 
It may be possible to deduplicate logic between `handler` and `handleAny`.

```go
func handleAny(any *codectypes.Any) error {
switch any.TypeURL {
case banktypes.MsgSend:
    msgResponse, err := unpackBankMsgSendResponse(any)
    if err != nil {
        return err
    }

    handleBankSendMsg(msgResponse)

case stakingtypes.MsgDelegate:
    msgResponse, err := unpackStakingDelegateResponse(any)
    if err != nil {
        return err
    }

    handleStakingDelegateMsg(msgResponse)

    case transfertypes.MsgTransfer:
    msgResponse, err := unpackIBCTransferMsgResponse(any)
    if err != nil {
        return err
    }

    handleIBCTransferMsg(msgResponse)
 
default:
    return
}
```