<!--
order: 5
-->

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

- `Owner` is an invalid bech32 address string.
- `ConnectionID` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).

This message will construct a new `MsgChannelOpenInit` on chain and route it to the core IBC message server to initiate the opening step of the channel handshake.

The `controller` module will generate a new port identifier and claim the associated port capability. The caller is expected to provide an appropriate application version string. For example, this may be an ICS27 JSON encoded `Metadata` type or an ICS29 JSON encoded `Metadata` type with a nested application version. 
If the `Version` string omitted the application will construct a default version string in the `OnChanOpenInit` handshake callback.

```go
type MsgRegisterInterchainAccountResponse struct {
  ChannelID string
}
```

The `ChannelID` is return in the message response.


## `MsgSendTx`

An Interchain Accounts transaction can be executed on a remote `host` chain by using `MsgSendTx` from the `controller` chain:

```go
type MsgSendTx struct {
  Owner           string
  ConnectionID    string
  PacketData      InterchainAccountPacketData 
  RelativeTimeout uint64
}
```

This message is expected to fail if:

- `Owner` is an invalid bech32 address string.
- `ConnectionID` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).
- `PacketData` contains an `UNSPECIFIED` type enum, the length of `Data` bytes is zero or the `Memo` field exceeds `256` characters in length.
- `RelativeTimeout` is zero.

This message will send the provided pre-built `PacketData` on the `controller` chain channel associated with the `Owner` and `ConnectionID`.
The `PacketData` is expected to contain a list of serialized `[]sdk.Msg` in the form of `CosmosTx`.
When the packet is relayed the `PacketData` is unmarshalled and the messages are executed on the receiving `host` chain.


```go
type MsgSendTxResponse struct {
  Sequence uint64
}
```

The packet `Sequence` is returned in the message response.
