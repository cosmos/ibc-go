---
title: Messages
sidebar_label: Messages
sidebar_position: 5
slug: /apps/interchain-accounts/messages
---


# Messages

## `MsgRegisterInterchainAccount`

An Interchain Accounts channel handshake can be initated using `MsgRegisterInterchainAccount`:

```go
type MsgRegisterInterchainAccount struct {
  Owner        string
  ConnectionID string
  Version      string
}
```

This message is expected to fail if:

- `Owner` is an empty string.
- `ConnectionID` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).

This message will construct a new `MsgChannelOpenInit` on chain and route it to the core IBC message server to initiate the opening step of the channel handshake.

The controller submodule will generate a new port identifier and claim the associated port capability. The caller is expected to provide an appropriate application version string. For example, this may be an ICS-27 JSON encoded [`Metadata`](https://github.com/cosmos/ibc-go/blob/v6.0.0/proto/ibc/applications/interchain_accounts/v1/metadata.proto#L11) type or an ICS-29 JSON encoded [`Metadata`](https://github.com/cosmos/ibc-go/blob/v6.0.0/proto/ibc/applications/fee/v1/metadata.proto#L11) type with a nested application version. 
If the `Version` string is omitted, the controller submodule will construct a default version string in the `OnChanOpenInit` handshake callback.

```go
type MsgRegisterInterchainAccountResponse struct {
  ChannelID string
  PortID string
}
```

The `ChannelID` and `PortID` are returned in the message response.

## `MsgSendTx`

An Interchain Accounts transaction can be executed on a remote host chain by sending a `MsgSendTx` from the corresponding controller chain:

```go
type MsgSendTx struct {
  Owner           string
  ConnectionID    string
  PacketData      InterchainAccountPacketData 
  RelativeTimeout uint64
}
```

This message is expected to fail if:

- `Owner` is an empty string.
- `ConnectionID` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).
- `PacketData` contains an `UNSPECIFIED` type enum, the length of `Data` bytes is zero or the `Memo` field exceeds 256 characters in length.
- `RelativeTimeout` is zero.

This message will create a new IBC packet with the provided `PacketData` and send it via the channel associated with the `Owner` and `ConnectionID`.
The `PacketData` is expected to contain a list of serialized `[]sdk.Msg` in the form of `CosmosTx`. Please note the signer field of each `sdk.Msg` must be the interchain account address. 
When the packet is relayed to the host chain, the `PacketData` is unmarshalled and the messages are authenticated and executed.

```go
type MsgSendTxResponse struct {
  Sequence uint64
}
```

The packet `Sequence` is returned in the message response.

## Atomicity

As the Interchain Accounts module supports the execution of multiple transactions using the Cosmos SDK `Msg` interface, it provides the same atomicity guarantees as Cosmos SDK-based applications, leveraging the [`CacheMultiStore`](https://docs.cosmos.network/main/learn/advanced/store#cachemultistore) architecture provided by the [`Context`](https://docs.cosmos.network/main/learn/advanced/context.html) type. 

This provides atomic execution of transactions when using Interchain Accounts, where state changes are only committed if all `Msg`s succeed.
