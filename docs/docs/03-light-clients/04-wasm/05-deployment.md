---
title: Deployment
sidebar_label: Deployment
sidebar_position: 5
slug: /ibc/light-clients/wasm/deployment
---

# Deployment

Learn how to upload a Wasm light client contract on a chain. {synopsis}

## Storing a new Wasm light client

Storage of Wasm light client contracts is permissioned (i.e. only allowed to an authority such as governance). The designated authority is specified when instantiating `08-wasm`'s keeper: both [`NewKeeperWithVM`](https://github.com/cosmos/ibc-go/blob/c95c22f45cb217d27aca2665af9ac60b0d2f3a0c/modules/light-clients/08-wasm/keeper/keeper.go#L33-L38) and [`NewKeeperWithConfig`](https://github.com/cosmos/ibc-go/blob/c95c22f45cb217d27aca2665af9ac60b0d2f3a0c/modules/light-clients/08-wasm/keeper/keeper.go#L52-L57) constructor functions accept an `authority` argument that must be the address of the authorized actor. For example, in `app.go`, when instantiating the keeper, you can pass the address of the governance module:

```go
// app.go
app.WasmClientKeeper = wasmkeeper.NewKeeperWithVM(
  appCodec,
  keys[wasmtypes.StoreKey],
  authtypes.NewModuleAddress(govtypes.ModuleName).String(), // authority
  wasmVM,
)
```

If governance is the allowed authority, the governance v1 proposal that needs to be submitted to upload a new light client contract should contain the message [`MsgStoreCode`](https://github.com/cosmos/ibc-go/blob/f822b4fa7932a657420aba219c563e06c4465221/proto/ibc/lightclients/wasm/v1/tx.proto#L16-L23) with the base64-encoded bytecode of the Wasm contract. Use the following CLI command and JSON as an example:

```shell
%s tx gov submit-proposal <path/to/proposal.json> --from <key_or_address>
```

where `proposal.json` contains:

```json
{
  "title": "Upload IBC Wasm light client",
  "summary": "Upload wasm client",
  "messages": [
    {
      "@type": "/ibc.lightclients.wasm.v1.MsgStoreCode",
      "signer": "cosmos1...", // the authority address (e.g. the gov module account address)
      "wasm_byte_code": "YWJ...PUB+" // standard base64 encoding of the Wasm contract byte code
    }
  ],
  "metadata": "AQ==",
  "deposit": "100stake"
}
```

To learn more about the `submit-proposal` CLI command, please check out [the relevant section in Cosmos SDK documentation](https://docs.cosmos.network/main/modules/gov#submit-proposal).

Alternatively, the process of submitting the proposal may be simpler if you use the CLI command `store-code`. This CLI command accepts as argument the file of the Wasm light client contract and takes care of constructing the proposal message with `MsgStoreCode` and broadcasting it. See section [`store-code`](./07-client.md#store-code) for more information.
