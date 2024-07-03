---
title: Migrations
sidebar_label: Migrations
sidebar_position: 9
slug: /ibc/light-clients/wasm/migrations
---

# Migrations

This guide provides instructions for migrating 08-wasm versions.

## From v0.2.0+ibc-go-v8.3-wasmvm-v2.0 to v0.3.0-ibc-go-v8.3-wasmvm-v2.0

### Contract developers

The `v0.3.0` release of 08-wasm for ibc-go `v8.3.x` and above introduces a breaking change for client contract developers.

The contract API `SudoMsg` payloads `VerifyMembershipMsg` and `VerifyNonMembershipMsg` have been modified. 
The encoding of the `Path` field of both structs has been updated from `v1.MerklePath` to `v2.MerklePath` to support proving values stored under keys which contain non-utf8 encoded symbols. 

As a result, the `Path` field now contains a `MerklePath` composed of `key_path` of `[][]byte` as opposed to `[]string`. The JSON field `path` containing `key_path` of both `VerifyMembershipMsg` and `VerifyNonMembershipMsg` structs will now marshal elements as base64 encoded bytestrings. See below for example JSON diff.

```diff
{
  "verify_membership": {
    "height": {
      "revision_height": 1
    },
    "delay_time_period": 0,
    "delay_block_period": 0,
    "proof":"dmFsaWQgcHJvb2Y=",
    "path": {
+      "key_path":["L2liYw==","L2tleS9wYXRo"]
-      "key_path":["/ibc","/key/path"]
    },
    "value":"dmFsdWU="
  }
}
```

A migration is required for existing 08-wasm client contracts in order to correctly handle the deserialisation of `key_path` from `[]string` to `[][]byte`.
Contract developers should familiarise themselves with the migration path offered by 08-wasm [here](./05-governance.md#migrating-an-existing-wasm-light-client-contract).

An example of the required changes in a client contract may look like:

```diff
#[cw_serde]
pub struct MerklePath {
+ #[schemars(with = "String")]
+ #[serde(with = "Base64", default)]
+ pub key_path: Vec<Bytes>,
- pub key_path: Vec<String>,
}
```

## From v0.1.1+ibc-go-v7.3-wasmvm-v1.5 to v0.2.0-ibc-go-v7.6-wasmvm-v1.5

### Contract developers

The `v0.2.0` release of 08-wasm for ibc-go `v7.6.x` and above introduces a breaking change for client contract developers.

The contract API `SudoMsg` payloads `VerifyMembershipMsg` and `VerifyNonMembershipMsg` have been modified. 
The encoding of the `Path` field of both structs has been updated from `v1.MerklePath` to `v2.MerklePath` to support proving values stored under keys which contain non-utf8 encoded symbols. 

As a result, the `Path` field now contains a `MerklePath` composed of `key_path` of `[][]byte` as opposed to `[]string`. The JSON field `path` containing `key_path` of both `VerifyMembershipMsg` and `VerifyNonMembershipMsg` structs will now marshal elements as base64 encoded bytestrings. See below for example JSON diff.

```diff
{
  "verify_membership": {
    "height": {
      "revision_height": 1
    },
    "delay_time_period": 0,
    "delay_block_period": 0,
    "proof":"dmFsaWQgcHJvb2Y=",
    "path": {
+      "key_path":["L2liYw==","L2tleS9wYXRo"]
-      "key_path":["/ibc","/key/path"]
    },
    "value":"dmFsdWU="
  }
}
```

A migration is required for existing 08-wasm client contracts in order to correctly handle the deserialisation of `key_path` from `[]string` to `[][]byte`.
Contract developers should familiarise themselves with the migration path offered by 08-wasm [here](./05-governance.md#migrating-an-existing-wasm-light-client-contract).

An example of the required changes in a client contract may look like:

```diff
#[cw_serde]
pub struct MerklePath {
+ #[schemars(with = "String")]
+ #[serde(with = "Base64", default)]
+ pub key_path: Vec<Bytes>,
- pub key_path: Vec<String>,
}
```

## From ibc-go v7.3.x to ibc-go v8.0.x

### Chains

In the 08-wasm versions compatible with ibc-go v7.3.x and above from the v7 release line, the checksums of the uploaded Wasm bytecodes are all stored under a single key. From ibc-go v8.0.x the checksums are stored using [`collections.KeySet`](https://docs.cosmos.network/v0.50/build/packages/collections#keyset), whose full functionality became available in Cosmos SDK v0.50. There is therefore an [automatic migration handler](https://github.com/cosmos/ibc-go/blob/57fcdb9a9a9db9b206f7df2f955866dc4e10fef4/modules/light-clients/08-wasm/module.go#L115-L118) configured in the 08-wasm module to migrate the stored checksums to `collections.KeySet`.

## From v0.1.0+ibc-go-v8.0-wasmvm-v1.5 to v0.2.0-ibc-go-v8.3-wasmvm-v2.0

The `WasmEngine` interface has been updated to reflect changes in the function signatures of Wasm VM:

```diff
type WasmEngine interface {
- StoreCode(code wasmvm.WasmCode) (wasmvm.Checksum, error)
+ StoreCode(code wasmvm.WasmCode, gasLimit uint64) (wasmvmtypes.Checksum, uint64, error)

  StoreCodeUnchecked(code wasmvm.WasmCode) (wasmvm.Checksum, error)

  Instantiate(
    checksum wasmvm.Checksum,
    env wasmvmtypes.Env,
    info wasmvmtypes.MessageInfo,
    initMsg []byte,
    store wasmvm.KVStore,
    goapi wasmvm.GoAPI,
    querier wasmvm.Querier,
    gasMeter wasmvm.GasMeter,
    gasLimit uint64,
    deserCost wasmvmtypes.UFraction,
- ) (*wasmvmtypes.Response, uint64, error)
+ ) (*wasmvmtypes.ContractResult, uint64, error)

  Query(
    checksum wasmvm.Checksum,
    env wasmvmtypes.Env,
    queryMsg []byte,
    store wasmvm.KVStore,
    goapi wasmvm.GoAPI,
    querier wasmvm.Querier,
    gasMeter wasmvm.GasMeter,
    gasLimit uint64,
    deserCost wasmvmtypes.UFraction,
- ) ([]byte, uint64, error)
+ ) (*wasmvmtypes.QueryResult, uint64, error)

  Migrate(
    checksum wasmvm.Checksum,
    env wasmvmtypes.Env,
    migrateMsg []byte,
    store wasmvm.KVStore,
    goapi wasmvm.GoAPI,
    querier wasmvm.Querier,
    gasMeter wasmvm.GasMeter,
    gasLimit uint64,
    deserCost wasmvmtypes.UFraction,
- ) (*wasmvmtypes.Response, uint64, error)
+ ) (*wasmvmtypes.ContractResult, uint64, error)

  Sudo(
    checksum wasmvm.Checksum,
    env wasmvmtypes.Env,
    sudoMsg []byte,
    store wasmvm.KVStore,
    goapi wasmvm.GoAPI,
    querier wasmvm.Querier,
    gasMeter wasmvm.GasMeter,
    gasLimit uint64,
    deserCost wasmvmtypes.UFraction,
- ) (*wasmvmtypes.Response, uint64, error)
+ ) (*wasmvmtypes.ContractResult, uint64, error)

  GetCode(checksum wasmvm.Checksum) (wasmvm.WasmCode, error)

  Pin(checksum wasmvm.Checksum) error

  Unpin(checksum wasmvm.Checksum) error
}
```

Similar changes were required in the functions of `MockWasmEngine` interface.

### Chains

The `SupportedCapabilities` field of `WasmConfig` is now of type `[]string`:

```diff
type WasmConfig struct {
  DataDir string
- SupportedCapabilities string
+ SupportedCapabilities []string
  ContractDebugMode bool
}
```
