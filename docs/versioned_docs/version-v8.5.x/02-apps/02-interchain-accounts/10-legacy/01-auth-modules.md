---
title: Authentication Modules
sidebar_label: Authentication Modules
sidebar_position: 1
slug: /apps/interchain-accounts/legacy/auth-modules
---


# Building an authentication module

## Deprecation Notice

**This document is deprecated and will be removed in future releases**.

:::note Synopsis
Authentication modules play the role of the `Base Application` as described in [ICS-30 IBC Middleware](https://github.com/cosmos/ibc/tree/master/spec/app/ics-030-middleware), and enable application developers to perform custom logic when working with the Interchain Accounts controller API.
:::

The controller submodule is used for account registration and packet sending. It executes only logic required of all controllers of interchain accounts. The type of authentication used to manage the interchain accounts remains unspecified. There may exist many different types of authentication which are desirable for different use cases. Thus the purpose of the authentication module is to wrap the controller submodule with custom authentication logic.

In ibc-go, authentication modules are connected to the controller chain via a middleware stack. The controller submodule is implemented as [middleware](https://github.com/cosmos/ibc/tree/master/spec/app/ics-030-middleware) and the authentication module is connected to the controller submodule as the base application of the middleware stack. To implement an authentication module, the `IBCModule` interface must be fulfilled. By implementing the controller submodule as middleware, any amount of authentication modules can be created and connected to the controller submodule without writing redundant code.

The authentication module must:

- Authenticate interchain account owners.
- Track the associated interchain account address for an owner.
- Send packets on behalf of an owner (after authentication).

> Please note that since ibc-go v6 the channel capability is claimed by the controller submodule and therefore it is not required for authentication modules to claim the capability in the `OnChanOpenInit` callback. When the authentication module sends packets on the channel created for the associated interchain account it can pass a `nil` capability to the legacy function `SendTx` of the controller keeper (see section [`SendTx`](03-keeper-api.md#sendtx) for more information).

## `IBCModule` implementation

The following `IBCModule` callbacks must be implemented with appropriate custom logic:

```go
// OnChanOpenInit implements the IBCModule interface
func (im IBCModule) OnChanOpenInit(
  ctx sdk.Context,
  order channeltypes.Order,
  connectionHops []string,
  portID string,
  channelID string,
  chanCap *capabilitytypes.Capability,
  counterparty channeltypes.Counterparty,
  version string,
) (string, error) {
  // since ibc-go v6 the authentication module *must not* claim the channel capability on OnChanOpenInit

  // perform custom logic

  return version, nil
}

// OnChanOpenAck implements the IBCModule interface
func (im IBCModule) OnChanOpenAck(
  ctx sdk.Context,
  portID,
  channelID string,
  counterpartyVersion string,
) error {
  // perform custom logic

  return nil
}

// OnChanCloseConfirm implements the IBCModule interface
func (im IBCModule) OnChanCloseConfirm(
  ctx sdk.Context,
  portID,
  channelID string,
) error {
  // perform custom logic

  return nil
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im IBCModule) OnAcknowledgementPacket(
  ctx sdk.Context,
  packet channeltypes.Packet,
  acknowledgement []byte,
  relayer sdk.AccAddress,
) error {
  // perform custom logic

  return nil
}

// OnTimeoutPacket implements the IBCModule interface.
func (im IBCModule) OnTimeoutPacket(
  ctx sdk.Context,
  packet channeltypes.Packet,
  relayer sdk.AccAddress,
) error {
  // perform custom logic

  return nil
}
```

The following functions must be defined to fulfill the `IBCModule` interface, but they will never be called by the controller submodule so they may error or panic. That is because in Interchain Accounts, the channel handshake is always initiated on the controller chain and packets are always sent to the host chain and never to the controller chain.

```go
// OnChanOpenTry implements the IBCModule interface
func (im IBCModule) OnChanOpenTry(
  ctx sdk.Context,
  order channeltypes.Order,
  connectionHops []string,
  portID,
  channelID string,
  chanCap *capabilitytypes.Capability,
  counterparty channeltypes.Counterparty,
  counterpartyVersion string,
) (string, error) {
  panic("UNIMPLEMENTED")
}

// OnChanOpenConfirm implements the IBCModule interface
func (im IBCModule) OnChanOpenConfirm(
  ctx sdk.Context,
  portID,
  channelID string,
) error {
  panic("UNIMPLEMENTED")
}

// OnChanCloseInit implements the IBCModule interface
func (im IBCModule) OnChanCloseInit(
  ctx sdk.Context,
  portID,
  channelID string,
) error {
  panic("UNIMPLEMENTED")
}

// OnRecvPacket implements the IBCModule interface. A successful acknowledgement
// is returned if the packet data is successfully decoded and the receive application
// logic returns without error.
func (im IBCModule) OnRecvPacket(
  ctx sdk.Context,
  packet channeltypes.Packet,
  relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
  panic("UNIMPLEMENTED")
}
```

## `OnAcknowledgementPacket`

Controller chains will be able to access the acknowledgement written into the host chain state once a relayer relays the acknowledgement.
The acknowledgement bytes contain either the response of the execution of the message(s) on the host chain or an error. They will be passed to the auth module via the `OnAcknowledgementPacket` callback. Auth modules are expected to know how to decode the acknowledgement.

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

If the `txMsgData.Data` field is non nil, the host chain is using SDK version \<\= v0.45.
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

## Integration into `app.go` file

To integrate the authentication module into your chain, please follow the steps outlined in [`app.go` integration](02-integration.md#example-integration).
