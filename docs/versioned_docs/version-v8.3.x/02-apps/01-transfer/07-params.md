---
title: Params
sidebar_label: Params
sidebar_position: 7
slug: /apps/transfer/params
---


# Parameters

The IBC transfer application module contains the following parameters:

| Name             | Type | Default Value |
| ---------------- | ---- | ------------- |
| `SendEnabled`    | bool | `true`        |
| `ReceiveEnabled` | bool | `true`        |

The IBC transfer module stores its parameters in its keeper with the prefix of `0x03`.

## `SendEnabled`

The `SendEnabled` parameter controls send cross-chain transfer capabilities for all fungible tokens.

To prevent a single token from being transferred from the chain, set the `SendEnabled` parameter to `true` and then, depending on the Cosmos SDK version, do one of the following:

- For Cosmos SDK v0.46.x or earlier, set the bank module's [`SendEnabled` parameter](https://github.com/cosmos/cosmos-sdk/blob/release/v0.46.x/x/bank/spec/05_params.md#sendenabled) for the denomination to `false`.
- For Cosmos SDK versions above v0.46.x, set the bank module's `SendEnabled` entry for the denomination to `false` using `MsgSetSendEnabled` as a governance proposal.

::: warning
Doing so will prevent the token from being transferred between any accounts in the blockchain.
:::

## `ReceiveEnabled`

The transfers enabled parameter controls receive cross-chain transfer capabilities for all fungible tokens.

To prevent a single token from being transferred to the chain, set the `ReceiveEnabled` parameter to `true` and then, depending on the Cosmos SDK version, do one of the following:

- For Cosmos SDK v0.46.x or earlier, set the bank module's [`SendEnabled` parameter](https://github.com/cosmos/cosmos-sdk/blob/release/v0.46.x/x/bank/spec/05_params.md#sendenabled) for the denomination to `false`.
- For Cosmos SDK versions above v0.46.x, set the bank module's `SendEnabled` entry for the denomination to `false` using `MsgSetSendEnabled` as a governance proposal.

::: warning
Doing so will prevent the token from being transferred between any accounts in the blockchain.
:::

## Queries

Current parameter values can be queried via a query message.

<!-- Turn it into a github code snippet in docusaurus: -->

```protobuf
// proto/ibc/applications/transfer/v1/query.proto

// QueryParamsRequest is the request type for the Query/Params RPC method.
message QueryParamsRequest {}

// QueryParamsResponse is the response type for the Query/Params RPC method.
message QueryParamsResponse {
  // params defines the parameters of the module.
  Params params = 1;
}
```

To execute the query in `simd`, you use the following command:

```bash
simd query ibc-transfer params
```

## Changing Parameters

To change the parameter values, you must make a governance proposal that executes the `MsgUpdateParams` message.

<!-- Turn it into a github code snippet in docusaurus: -->

```protobuf
// proto/ibc/applications/transfer/v1/tx.proto

// MsgUpdateParams is the Msg/UpdateParams request type.
message MsgUpdateParams {
  // signer address (it may be the address that controls the module, which defaults to x/gov unless overwritten).
  string signer = 1;

  // params defines the transfer parameters to update.
  //
  // NOTE: All parameters must be supplied.
  Params params = 2 [(gogoproto.nullable) = false];
}

// MsgUpdateParamsResponse defines the response structure for executing a
// MsgUpdateParams message.
message MsgUpdateParamsResponse {}
```
