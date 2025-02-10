---
title: Governance
sidebar_label: Governance
sidebar_position: 5
slug: /ibc/light-clients/wasm/governance
---

# Governance

Learn how to upload Wasm light client byte code on a chain, and how to migrate an existing Wasm light client contract. 

## Setting an authority

Both the storage of Wasm light client byte code as well as the migration of an existing Wasm light client contract are permissioned (i.e. only allowed to an authority such as governance). The designated authority is specified when instantiating `08-wasm`'s keeper: both [`NewKeeperWithVM`](https://github.com/cosmos/ibc-go/blob/57fcdb9a9a9db9b206f7df2f955866dc4e10fef4/modules/light-clients/08-wasm/keeper/keeper.go#L39-L47) and [`NewKeeperWithConfig`](https://github.com/cosmos/ibc-go/blob/57fcdb9a9a9db9b206f7df2f955866dc4e10fef4/modules/light-clients/08-wasm/keeper/keeper.go#L88-L96) constructor functions accept an `authority` argument that must be the address of the authorized actor. For example, in `app.go`, when instantiating the keeper, you can pass the address of the governance module:

```go
// app.go
import (
  ...
  "github.com/cosmos/cosmos-sdk/runtime"
  authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
  govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

  ibcwasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
  ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
  ...
)

// app.go
app.WasmClientKeeper = ibcwasmkeeper.NewKeeperWithVM(
  appCodec,
  runtime.NewKVStoreService(keys[ibcwasmtypes.StoreKey]),
  app.IBCKeeper.ClientKeeper,
 	authtypes.NewModuleAddress(govtypes.ModuleName).String(), // authority
  wasmVM,
  app.GRPCQueryRouter(),
)
```

## Storing new Wasm light client byte code

 If governance is the allowed authority, the governance v1 proposal that needs to be submitted to upload a new light client contract should contain the message [`MsgStoreCode`](https://github.com/cosmos/ibc-go/blob/57fcdb9a9a9db9b206f7df2f955866dc4e10fef4/proto/ibc/lightclients/wasm/v1/tx.proto#L23-L30) with the base64-encoded byte code of the Wasm contract. Use the following CLI command and JSON as an example:

```shell
simd tx gov submit-proposal <path/to/proposal.json> --from <key_or_address>
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

Alternatively, the process of submitting the proposal may be simpler if you use the CLI command `store-code`. This CLI command accepts as argument the file of the Wasm light client contract and takes care of constructing the proposal message with `MsgStoreCode` and broadcasting it. See section [`store-code`](./08-client.md#store-code) for more information.

## Migrating an existing Wasm light client contract

If governance is the allowed authority, the governance v1 proposal that needs to be submitted to migrate an existing new Wasm light client contract should contain the message [`MsgMigrateContract`](https://github.com/cosmos/ibc-go/blob/57fcdb9a9a9db9b206f7df2f955866dc4e10fef4/proto/ibc/lightclients/wasm/v1/tx.proto#L52-L63) with the checksum of the Wasm byte code to migrate to. Use the following CLI command and JSON as an example:

```shell
simd tx gov submit-proposal <path/to/proposal.json> --from <key_or_address>
```

where `proposal.json` contains:

```json
{
  "title": "Migrate IBC Wasm light client",
  "summary": "Migrate wasm client",
  "messages": [
    {
      "@type": "/ibc.lightclients.wasm.v1.MsgMigrateContract",
      "signer": "cosmos1...", // the authority address (e.g. the gov module account address)
      "client_id": "08-wasm-1", // client identifier of the Wasm light client contract that will be migrated
      "checksum": "a8ad...4dc0", // SHA-256 hash of the Wasm byte code to migrate to, previously stored with MsgStoreCode
      "msg": "{}" // JSON-encoded message to be passed to the contract on migration
    }
  ],
  "metadata": "AQ==",
  "deposit": "100stake"
}
```

To learn more about the `submit-proposal` CLI command, please check out [the relevant section in Cosmos SDK documentation](https://docs.cosmos.network/main/modules/gov#submit-proposal).

## Removing an existing checksum

If governance is the allowed authority, the governance v1 proposal that needs to be submitted to remove a specific checksum from the list of allowed checksums should contain the message [`MsgRemoveChecksum`](https://github.com/cosmos/ibc-go/blob/57fcdb9a9a9db9b206f7df2f955866dc4e10fef4/proto/ibc/lightclients/wasm/v1/tx.proto#L39-L46) with the checksum (of a corresponding Wasm byte code). Use the following CLI command and JSON as an example:

```shell
simd tx gov submit-proposal <path/to/proposal.json> --from <key_or_address>
```

where `proposal.json` contains:

```json
{
  "title": "Remove checksum of Wasm light client byte code",
  "summary": "Remove checksum",
  "messages": [
    {
      "@type": "/ibc.lightclients.wasm.v1.MsgRemoveChecksum",
      "signer": "cosmos1...", // the authority address (e.g. the gov module account address)
      "checksum": "a8ad...4dc0", // SHA-256 hash of the Wasm byte code that should be removed from the list of allowed checksums
    }
  ],
  "metadata": "AQ==",
  "deposit": "100stake"
}
```

To learn more about the `submit-proposal` CLI command, please check out [the relevant section in Cosmos SDK documentation](https://docs.cosmos.network/main/modules/gov#submit-proposal).
