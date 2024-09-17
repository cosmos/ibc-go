---
title: Client
sidebar_label: Client
sidebar_position: 7
slug: /apps/interchain-accounts/client
---


# Client

## CLI

A user can query and interact with the Interchain Accounts module using the CLI. Use the `--help` flag to discover the available commands:

```shell
simd query interchain-accounts --help
```

> Please not that this section does not document all the available commands, but only the ones that deserved extra documentation that was not possible to fit in the command line documentation.

### Controller

A user can query and interact with the controller submodule.

#### Query

The `query` commands allow users to query the controller submodule.

```shell
simd query interchain-accounts controller --help
```

#### Transactions

The `tx` commands allow users to interact with the controller submodule.

```shell
simd tx interchain-accounts controller --help
```

#### `send-tx`

The `send-tx` command allows users to send a transaction on the provided connection to be executed using an interchain account on the host chain.

```shell
simd tx interchain-accounts controller send-tx [connection-id] [path/to/packet_msg.json]
```

Example:

```shell
simd tx interchain-accounts controller send-tx connection-0 packet-data.json --from cosmos1..
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

A helper CLI is provided in the host submodule which can be used to generate the packet data JSON using the counterparty chain's binary. See the [`generate-packet-data` command](#generate-packet-data) for an example.

### Host

A user can query and interact with the host submodule.

#### Query

The `query` commands allow users to query the host submodule.

```shell
simd query interchain-accounts host --help
```

#### Transactions

The `tx` commands allow users to interact with the controller submodule.

```shell
simd tx interchain-accounts host --help
```

##### `generate-packet-data`

The `generate-packet-data` command allows users to generate interchain accounts packet data for input message(s). The packet data can then be used with the controller submodule's [`send-tx` command](#send-tx).

```shell
simd tx interchain-accounts host generate-packet-data [message]
```

Example:

```shell
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
}]' --memo memo
```

The command accepts a single `sdk.Msg` or a list of `sdk.Msg`s that will be encoded into the outputs `data` field.

Example output:

```json
{
  "type":"TYPE_EXECUTE_TX",
  "data":"CqIBChwvY29zbW9zLmJhbmsudjFiZXRhMS5Nc2dTZW5kEoEBCkFjb3Ntb3MxNWNjc2hobXAwZ3N4MjlxcHFxNmc0em1sdG5udmdteXU5dWV1YWRoOXkybmM1emowc3psczVndGRkehItY29zbW9zMTBoOXN0YzV2Nm50Z2V5Z2Y1eGY5NDVuanFxNWgzMnI1M3VxdXZ3Gg0KBXN0YWtlEgQxMDAw",
  "memo":"memo"
}
```

## gRPC

A user can query the interchain account module using gRPC endpoints.

### Controller 

A user can query the controller submodule using gRPC endpoints.

#### `InterchainAccount`

The `InterchainAccount` endpoint allows users to query the controller submodule for the interchain account address for a given owner on a particular connection.

```shell
ibc.applications.interchain_accounts.controller.v1.Query/InterchainAccount
```

Example:

```shell
grpcurl -plaintext \
    -d '{"owner":"cosmos1..","connection_id":"connection-0"}' \
    localhost:9090 \
    ibc.applications.interchain_accounts.controller.v1.Query/InterchainAccount
```

#### `Params`

The `Params` endpoint users to query the current controller submodule parameters.

```shell
ibc.applications.interchain_accounts.controller.v1.Query/Params
```

Example:

```shell
grpcurl -plaintext \
    localhost:9090 \
    ibc.applications.interchain_accounts.controller.v1.Query/Params
```

### Host 

A user can query the host submodule using gRPC endpoints.

#### `Params`

The `Params` endpoint users to query the current host submodule parameters.

```shell
ibc.applications.interchain_accounts.host.v1.Query/Params
```

Example:

```shell
grpcurl -plaintext \
    localhost:9090 \
    ibc.applications.interchain_accounts.host.v1.Query/Params
```
