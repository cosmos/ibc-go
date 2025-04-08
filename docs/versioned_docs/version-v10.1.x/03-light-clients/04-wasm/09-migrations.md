---
title: Migrations
sidebar_label: Migrations
sidebar_position: 9
slug: /ibc/light-clients/wasm/migrations
---

# Migrations

This guide provides instructions for migrating 08-wasm versions.

Please note that the following releases are retracted. Please refer to the appropriate migrations section for upgrading.

```bash
v0.3.1-0.20240717085919-bb71eef0f3bf => v0.3.0+ibc-go-v8.3-wasmvm-v2.0
v0.2.1-0.20240717085554-570d057959e3 => v0.2.0+ibc-go-v7.6-wasmvm-v1.5
v0.2.1-0.20240523101951-4b45d1822fb6 => v0.2.0+ibc-go-v8.3-wasmvm-v2.0
v0.1.2-0.20240412103620-7ee2a2452b79 => v0.1.1+ibc-go-v7.3-wasmvm-v1.5
v0.1.1-0.20231213092650-57fcdb9a9a9d => v0.1.0+ibc-go-v8.0-wasmvm-v1.5
v0.1.1-0.20231213092633-b306e7a706e1 => v0.1.0+ibc-go-v7.3-wasmvm-v1.5
```

## From ibc-go v8.4.x to ibc-go v9.0.x

### Chains

- The `Initialize`, `Status`, `GetTimestampAtHeight`, `GetLatestHeight`, `VerifyMembership`, `VerifyNonMembership`, `VerifyClientMessage`, `UpdateState` and `UpdateStateOnMisbehaviour` functions in `ClientState` have been removed and all their logic has been moved to functions of the `LightClientModule`.
- The `MigrateContract` function has been removed from `ClientState`.
- The `VerifyMembershipMsg` and `VerifyNonMembershipMsg` payloads for `SudoMsg` have been modified. The `Path` field of both structs has been updated from `v1.MerklePath` to `v2.MerklePath`. The new `v2.MerklePath` field contains a `KeyPath` of `[][]byte` as opposed to `[]string`, see [23-commitment](../../05-migrations/13-v8-to-v9.md#23-commitment). This supports proving values stored under keys which contain non-utf8 encoded symbols. As a result, the JSON field `path` containing `key_path` of both messages will marshal elements as a base64 encoded bytestrings. This is a breaking change for 08-wasm client contracts and they should be migrated to correctly support deserialisation of the `v2.MerklePath` field.
- The `ExportMetadataMsg` struct has been removed and is no longer required for contracts to implement. Core IBC will handle exporting all key/value's written to the store by a light client contract.
- The `ZeroCustomFields` interface function has been removed from the `ClientState` interface. Core IBC only used this function to set tendermint client states when scheduling an IBC software upgrade. The interface function has been replaced by a type assertion.
- The `MaxWasmByteSize` function has been removed in favor of the `MaxWasmSize` constant.
- The `HasChecksum`, `GetAllChecksums` and `Logger` functions have been moved from the `types` package to a method on the `Keeper` type in the `keeper` package.
- The `InitializePinnedCodes` function has been moved to a method on the `Keeper` type in the `keeper` package.
- The `CustomQuerier`, `StargateQuerier` and `QueryPlugins` types have been moved from the `types` package to the `keeper` package.
- The `NewDefaultQueryPlugins`, `AcceptListStargateQuerier` and `RejectCustomQuerier` functions has been moved from the `types` package to the `keeper` package.
- The `NewDefaultQueryPlugins` function signature has changed to take an argument: `queryRouter ibcwasm.QueryRouter`.
- The `AcceptListStargateQuerier` function signature has changed to take an additional argument: `queryRouter ibcwasm.QueryRouter`.
- The `WithQueryPlugins` function signature has changed to take in the `QueryPlugins` type from the `keeper` package (previously from the `types` package).
- The `VMGasRegister` variable has been moved from the `types` package to the `keeper` package.

## From v0.3.0+ibc-go-v8.3-wasmvm-v2.0 to v0.4.1-ibc-go-v8.4-wasmvm-v2.0

### Contract developers

Contract developers are required to update their JSON API message structure for the `SudoMsg` payloads `VerifyMembershipMsg` and `VerifyNonMembershipMsg`.
The `path` field on both JSON API messages has been renamed to `merkle_path`.

A migration is required for existing 08-wasm client contracts in order to correctly handle the deserialisation of these fields.

## From v0.2.0+ibc-go-v7.3-wasmvm-v1.5 to v0.3.1-ibc-go-v7.4-wasmvm-v1.5

### Contract developers

Contract developers are required to update their JSON API message structure for the `SudoMsg` payloads `VerifyMembershipMsg` and `VerifyNonMembershipMsg`.
The `path` field on both JSON API messages has been renamed to `merkle_path`.

A migration is required for existing 08-wasm client contracts in order to correctly handle the deserialisation of these fields.

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
+   pub key_path: Vec<cosmwasm_std::Binary>,
-   pub key_path: Vec<String>,
}
```

Please refer to the [`cosmwasm_std`](https://docs.rs/cosmwasm-std/2.0.4/cosmwasm_std/struct.Binary.html) documentation for more information.

## From v0.1.1+ibc-go-v7.3-wasmvm-v1.5 to v0.2.0-ibc-go-v7.3-wasmvm-v1.5

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
+   pub key_path: Vec<cosmwasm_std::Binary>,
-   pub key_path: Vec<String>,
}
```

Please refer to the [`cosmwasm_std`](https://docs.rs/cosmwasm-std/2.0.4/cosmwasm_std/struct.Binary.html) documentation for more information.

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
