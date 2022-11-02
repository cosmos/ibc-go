<!--
order: 3
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

- `Owner` is an empty string.
- `ConnectionID` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).

This message will construct a new `MsgChannelOpenInit` on chain and route it to the core IBC message server to initiate the opening step of the channel handshake.

The controller module will generate a new port identifier and claim the associated port capability. The caller is expected to provide an appropriate application version string. For example, this may be an ICS27 JSON encoded [`Metadata`](https://github.com/cosmos/ibc-go/blob/v6.0.0-alpha1/proto/ibc/applications/interchain_accounts/v1/metadata.proto#L11) type or an ICS29 JSON encoded [`Metadata`](https://github.com/cosmos/ibc-go/blob/v6.0.0-alpha1/proto/ibc/applications/fee/v1/metadata.proto#L11) type with a nested application version. 
If the `Version` string is omitted,  the application will construct a default version string in the `OnChanOpenInit` handshake callback.

```go
type MsgRegisterInterchainAccountResponse struct {
  ChannelID string
}
```

The `ChannelID` is return in the message response.

### CLI

The following is an example usage of the controller CLI command used to register an interchain account.

```bash
simd tx interchain-accounts controller register connection-0 --from cosmos1m9l358xunhhwds0568za49mzhvuxx9uxre5tud
```

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

### CLI

The following is an example usage of the controller CLI command used to send a transaction to be executed using an interchain account on the corresponding host chain.

```bash
simd tx interchain-accounts controller send-tx connection-0 packet-data.json --from cosmos1m9l358xunhhwds0568za49mzhvuxx9uxre5tud
```

See below for example contents of `packet-data.json`. The CLI handler will unmarshal the following into `InterchainAccountPacketData` appropriately.

```json
{
  "type":"TYPE_EXECUTE_TX",
  "data":"CqIBChwvY29zbW9zLmJhbmsudjFiZXRhMS5Nc2dTZW5kEoEBCkFjb3Ntb3MxNWNjc2hobXAwZ3N4MjlxcHFxNmc0em1sdG5udmdteXU5dWV1YWRoOXkybmM1emowc3psczVndGRkehItY29zbW9zMTBoOXN0YzV2Nm50Z2V5Z2Y1eGY5NDVuanFxNWgzMnI1M3VxdXZ3Gg0KBXN0YWtlEgQxMDAw",
  "memo":""
}
```

Note the `data` field is a base64 encoded byte string as per the [proto3 JSON encoding specification](https://developers.google.com/protocol-buffers/docs/proto3#json).

A helper CLI is provided in the host submodule which can be used to generate the packet data JSON using the counterparty chain's binary.
It accepts a list of `sdk.Msg`s which will be encoded into the outputs `data` field.

```bash
simd tx interchain-accounts host generate-packet-data '[{
    "@type":"/cosmos.bank.v1beta1.MsgSend",
    "from_address":"cosmos15ccshhmp0gsx29qpqq6g4zmltnnvgmyu9ueuadh9y2nc5zj0szls5gtddz",
    "to_address":"cosmos10h9stc5v6ntgeygf5xf945njqq5h32r53uquvw",
    "amount": [
        {
            "denom": "stake",
            "amount": "1000"
        }
    ]
}]'
```

The host submodule also provides a helper CLI to inspect the events of interchain accounts packets by providing the channel ID and packet sequence:

```bash
 simd q interchain-accounts host packet-events channel-0 100
```

### Atomicity

As the Interchain Accounts module supports the execution of multiple transactions using the Cosmos SDK `Msg` interface, it provides the same atomicity guarantees as Cosmos SDK-based applications, leveraging the [`CacheMultiStore`](https://docs.cosmos.network/main/core/store.html#cachemultistore) architecture provided by the [`Context`](https://docs.cosmos.network/main/core/context.html) type. 

This provides atomic execution of transactions when using Interchain Accounts, where state changes are only committed if all `Msg`s succeed.
